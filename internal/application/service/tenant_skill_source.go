package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/utils"
)

const (
	maxTenantSkillSourceURLLength = 2048
	tenantSkillSourceTimeout      = 5 * time.Minute
	tenantSkillSourceMaxRedirects = 3
	tenantSkillSourceMaxHops      = 3
	defaultTenantSkillRegistry    = "https://clawhub.ai"
	skillsShSourcePrefix          = "skills-sh:"
)

var tenantSkillSemverLike = regexp.MustCompile(`^v?\d+\.\d+(\.\d+)?([.-][0-9A-Za-z.-]+)?$`)

var ErrSkillSourceInvalid = errors.New("skill source is invalid")

const (
	tenantSkillSourceRegistry = "registry"
	tenantSkillSourceSkillsSh = "skills-sh"
	tenantSkillSourceGitHub   = "github"
	tenantSkillSourceGitLab   = "gitlab"
	tenantSkillSourceDirect   = "direct"
)

// tenantSkillSource is a normalized locator. Ref preserves the user-facing
// source for audit/API responses; GitRef and Subdir describe a repository
// archive when the source is a GitHub/GitLab tree.
type tenantSkillSource struct {
	Kind      string
	Ref       string
	URL       *url.URL
	Registry  string
	Slug      string
	Version   string
	Owner     string
	Repo      string
	GitRef    string
	Subdir    string
	DirectURL string
}

type tenantSkillSourceHandoff struct {
	OK          *bool                      `json:"ok"`
	Message     string                     `json:"message"`
	Reason      string                     `json:"reason"`
	InstallKind string                     `json:"installKind"`
	SourceRef   string                     `json:"sourceRef"`
	Repo        string                     `json:"repo"`
	Commit      string                     `json:"commit"`
	Path        string                     `json:"path"`
	ArchiveURL  string                     `json:"archiveUrl"`
	DownloadURL string                     `json:"downloadUrl"`
	GitHub      *tenantSkillGitHubHandoff  `json:"github"`
	Archive     *tenantSkillArchiveHandoff `json:"archive"`
}

type tenantSkillGitHubHandoff struct {
	Repo      string `json:"repo"`
	Path      string `json:"path"`
	Commit    string `json:"commit"`
	SourceURL string `json:"sourceUrl"`
}

type tenantSkillArchiveHandoff struct {
	Version     string `json:"version"`
	DownloadURL string `json:"downloadUrl"`
}

type fetchedTenantSkillSource struct {
	body        []byte
	contentType string
	subdir      string
	registry    string
}

func parseTenantSkillSource(raw string) (tenantSkillSource, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return tenantSkillSource{}, fmt.Errorf("%w: source is required", ErrSkillSourceInvalid)
	}
	if len(input) > maxTenantSkillSourceURLLength {
		return tenantSkillSource{}, fmt.Errorf("%w: source is too long", ErrSkillSourceInvalid)
	}
	input = strings.TrimRight(input, "/")

	if strings.Contains(input, "://") {
		return parseTenantSkillSourceURL(input)
	}
	if payload, ok := skillsShBarePayload(input); ok {
		return parseSkillsShLocator(defaultTenantSkillRegistry, splitTenantSkillPath(payload))
	}
	if strings.HasPrefix(input, "@") {
		return parseTenantSkillRegistrySlug(defaultTenantSkillRegistry, strings.TrimPrefix(input, "@"))
	}
	if strings.Contains(input, "/") {
		return tenantSkillSource{}, fmt.Errorf(
			"%w: %q is ambiguous; use @%s for ClawHub, or paste a GitHub, GitLab, skills.sh, or SkillHub URL",
			ErrSkillSourceInvalid, input, input,
		)
	}
	return parseTenantSkillRegistrySlug(defaultTenantSkillRegistry, input)
}

func parseTenantSkillSourceURL(raw string) (tenantSkillSource, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return tenantSkillSource{}, fmt.Errorf("%w: invalid URL", ErrSkillSourceInvalid)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return tenantSkillSource{}, fmt.Errorf("%w: only http(s) sources are allowed", ErrSkillSourceInvalid)
	}
	if parsed.Hostname() == "" {
		return tenantSkillSource{}, fmt.Errorf("%w: source URL has no hostname", ErrSkillSourceInvalid)
	}
	if parsed.User != nil {
		return tenantSkillSource{}, fmt.Errorf("%w: source URL cannot contain credentials", ErrSkillSourceInvalid)
	}
	if hasSensitiveSourceQuery(parsed.Query()) {
		return tenantSkillSource{}, fmt.Errorf("%w: source URL must not contain access credentials", ErrSkillSourceInvalid)
	}
	if err := utils.ValidateURLForSSRF(parsed.String()); err != nil {
		return tenantSkillSource{}, fmt.Errorf("%w: source URL rejected: %v", ErrSkillSourceInvalid, err)
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Fragment = ""
	ref := parsed.String()
	host := strings.ToLower(parsed.Hostname())

	switch {
	case host == "github.com" || host == "www.github.com" || host == "codeload.github.com":
		if host == "codeload.github.com" || isGitHubArchivePath(parsed.Path) {
			return tenantSkillSource{
				Kind: tenantSkillSourceGitHub, Ref: ref, URL: parsed, DirectURL: ref,
			}, nil
		}
		return parseTenantSkillGitHubURL(parsed, ref)
	case host == "gitlab.com" || host == "www.gitlab.com":
		if isGitLabArchivePath(parsed.Path) {
			return tenantSkillSource{
				Kind: tenantSkillSourceGitLab, Ref: ref, URL: parsed, DirectURL: ref,
			}, nil
		}
		return parseTenantSkillGitLabURL(parsed, ref)
	case host == "skills.sh" || host == "www.skills.sh":
		return parseTenantSkillSkillsShURL(parsed, ref)
	case isClawHubHost(host):
		return parseTenantSkillRegistryURL(parsed, ref)
	case isSkillHubCNHost(host):
		return parseTenantSkillSkillHubCNURL(parsed, ref)
	default:
		if isDirectSkillPath(parsed.Path) {
			return tenantSkillSource{
				Kind: tenantSkillSourceDirect, Ref: ref, URL: parsed, DirectURL: ref,
			}, nil
		}
		return parseTenantSkillRegistryURL(parsed, ref)
	}
}

func isClawHubHost(host string) bool {
	switch strings.ToLower(host) {
	case "clawhub.ai", "www.clawhub.ai", "clawhub.com", "www.clawhub.com":
		return true
	default:
		return false
	}
}

const tenantSkillHubCNAPIOrigin = "https://api.skillhub.cn"

func isSkillHubCNHost(host string) bool {
	switch strings.ToLower(host) {
	case "skillhub.cn", "www.skillhub.cn", "api.skillhub.cn":
		return true
	default:
		return false
	}
}

func parseTenantSkillSkillHubCNURL(u *url.URL, ref string) (tenantSkillSource, error) {
	trimmed := strings.Trim(u.Path, "/")
	if strings.HasPrefix(trimmed, "api/v1/download") {
		downloadURL := tenantSkillHubCNAPIOrigin + "/api/v1/download"
		if u.RawQuery != "" {
			downloadURL += "?" + u.RawQuery
		}
		return tenantSkillSource{
			Kind: tenantSkillSourceRegistry, Ref: ref, URL: u,
			Registry: tenantSkillHubCNAPIOrigin, DirectURL: downloadURL,
		}, nil
	}
	parts := splitTenantSkillPath(trimmed)
	if len(parts) > 0 && strings.EqualFold(parts[0], "skills") {
		parts = parts[1:]
	}
	if len(parts) == 0 || len(parts) > 2 {
		return tenantSkillSource{}, fmt.Errorf("%w: unrecognized SkillHub path", ErrSkillSourceInvalid)
	}
	slug := parts[len(parts)-1]
	slug, version := splitTenantSkillVersion(slug)
	if queryVersion := strings.TrimSpace(u.Query().Get("version")); queryVersion != "" {
		version = queryVersion
	}
	if slug == "" {
		return tenantSkillSource{}, fmt.Errorf("%w: skill slug is required", ErrSkillSourceInvalid)
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceRegistry, Ref: ref, URL: u,
		Registry: tenantSkillHubCNAPIOrigin, Slug: slug, Version: version,
	}, nil
}

func parseTenantSkillRegistryURL(u *url.URL, ref string) (tenantSkillSource, error) {
	origin := u.Scheme + "://" + u.Host
	trimmed := strings.Trim(u.Path, "/")
	if strings.HasPrefix(trimmed, "api/v1/download") {
		return tenantSkillSource{
			Kind: tenantSkillSourceRegistry, Ref: ref, URL: u,
			Registry: origin, DirectURL: ref,
		}, nil
	}
	parts := splitTenantSkillPath(trimmed)
	if len(parts) > 0 && strings.EqualFold(parts[0], "skills-sh") {
		return parseSkillsShLocator(origin, parts[1:])
	}
	slug, version, err := tenantSkillSlugAndVersion(trimmed, u.Fragment)
	if err != nil {
		return tenantSkillSource{}, err
	}
	if queryVersion := strings.TrimSpace(u.Query().Get("version")); queryVersion != "" {
		version = queryVersion
	}
	owner := ""
	if isClawHubHost(u.Hostname()) {
		owner, slug = splitTenantSkillOwnerSlug(slug)
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceRegistry, Ref: ref, URL: u,
		Registry: origin, Owner: owner, Slug: slug, Version: version,
	}, nil
}

func parseTenantSkillRegistrySlug(origin, spec string) (tenantSkillSource, error) {
	spec = strings.TrimSpace(strings.TrimPrefix(spec, "@"))
	if spec == "" {
		return tenantSkillSource{}, fmt.Errorf("%w: skill slug is required", ErrSkillSourceInvalid)
	}
	slug, version := splitTenantSkillVersion(spec)
	if slug == "" {
		return tenantSkillSource{}, fmt.Errorf("%w: skill slug is required", ErrSkillSourceInvalid)
	}
	owner := ""
	if isClawHubOrigin(origin) {
		owner, slug = splitTenantSkillOwnerSlug(slug)
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceRegistry, Ref: origin + "/" + spec,
		Registry: origin, Owner: owner, Slug: slug, Version: version,
	}, nil
}

func parseTenantSkillGitHubURL(u *url.URL, ref string) (tenantSkillSource, error) {
	parts := splitTenantSkillPath(u.Path)
	if len(parts) < 2 {
		return tenantSkillSource{}, fmt.Errorf("%w: GitHub URL must be owner/repo", ErrSkillSourceInvalid)
	}
	source := tenantSkillSource{
		Kind: tenantSkillSourceGitHub, Ref: ref,
		URL: u, Owner: parts[0], Repo: strings.TrimSuffix(parts[1], ".git"), GitRef: "HEAD",
	}
	rest := parts[2:]
	if len(rest) == 0 {
		return source, nil
	}
	switch rest[0] {
	case "tree", "blob":
		if len(rest) < 2 {
			return tenantSkillSource{}, fmt.Errorf("%w: GitHub URL is missing a ref", ErrSkillSourceInvalid)
		}
		source.GitRef = rest[1]
		source.Subdir = strings.Join(rest[2:], "/")
		if rest[0] == "blob" && strings.EqualFold(path.Base(source.Subdir), skills.SkillFileName) {
			source.Subdir = path.Dir(source.Subdir)
			if source.Subdir == "." {
				source.Subdir = ""
			}
		}
	case "archive", "releases":
		source.Kind = tenantSkillSourceDirect
		source.DirectURL = ref
	default:
		source.Subdir = strings.Join(rest, "/")
	}
	return source, nil
}

func parseTenantSkillGitLabURL(u *url.URL, ref string) (tenantSkillSource, error) {
	trimmed := strings.Trim(u.Path, "/")
	project, extra, found := strings.Cut(trimmed, "/-/")
	if !found {
		project = trimmed
	}
	parts := splitTenantSkillPath(project)
	if len(parts) < 2 {
		return tenantSkillSource{}, fmt.Errorf("%w: GitLab URL must be group/project", ErrSkillSourceInvalid)
	}
	source := tenantSkillSource{
		Kind: tenantSkillSourceGitLab, Ref: ref, URL: u,
		Owner:  strings.Join(parts[:len(parts)-1], "/"),
		Repo:   strings.TrimSuffix(parts[len(parts)-1], ".git"),
		GitRef: "HEAD",
	}
	if extra == "" {
		return source, nil
	}
	extraParts := splitTenantSkillPath(extra)
	if len(extraParts) == 0 {
		return source, nil
	}
	switch extraParts[0] {
	case "tree", "blob":
		if len(extraParts) < 2 {
			return tenantSkillSource{}, fmt.Errorf("%w: GitLab URL is missing a ref", ErrSkillSourceInvalid)
		}
		source.GitRef = extraParts[1]
		source.Subdir = strings.Join(extraParts[2:], "/")
		if extraParts[0] == "blob" && strings.EqualFold(path.Base(source.Subdir), skills.SkillFileName) {
			source.Subdir = path.Dir(source.Subdir)
			if source.Subdir == "." {
				source.Subdir = ""
			}
		}
	case "archive":
		source.Kind = tenantSkillSourceDirect
		source.DirectURL = ref
	}
	return source, nil
}

func parseTenantSkillSkillsShURL(u *url.URL, ref string) (tenantSkillSource, error) {
	parts := splitTenantSkillPath(u.Path)
	if len(parts) < 2 {
		return tenantSkillSource{}, fmt.Errorf("%w: skills.sh URL must be owner/repo", ErrSkillSourceInvalid)
	}
	if len(parts) >= 3 {
		return parseSkillsShLocator(defaultTenantSkillRegistry, parts)
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceGitHub, Ref: ref, URL: u,
		Owner: parts[0], Repo: strings.TrimSuffix(parts[1], ".git"), GitRef: "HEAD",
	}, nil
}

func skillsShBarePayload(input string) (string, bool) {
	lower := strings.ToLower(input)
	switch {
	case strings.HasPrefix(lower, skillsShSourcePrefix):
		return strings.TrimSpace(input[len(skillsShSourcePrefix):]), true
	case strings.HasPrefix(lower, "skills-sh/"):
		return strings.TrimSpace(input[len("skills-sh/"):]), true
	default:
		return "", false
	}
}

func parseSkillsShLocator(registry string, parts []string) (tenantSkillSource, error) {
	if len(parts) != 3 {
		return tenantSkillSource{}, fmt.Errorf(
			"%w: skills.sh locator must be owner/repo/slug", ErrSkillSourceInvalid,
		)
	}
	owner := strings.ToLower(strings.TrimSpace(parts[0]))
	repo := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git"))
	slug := strings.ToLower(strings.TrimSpace(parts[2]))
	slug, _ = splitTenantSkillVersion(slug)
	if owner == "" || repo == "" || slug == "" ||
		strings.Contains(owner, "..") || strings.Contains(repo, "..") || strings.Contains(slug, "..") {
		return tenantSkillSource{}, fmt.Errorf(
			"%w: skills.sh locator must be owner/repo/slug", ErrSkillSourceInvalid,
		)
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceSkillsSh, Ref: skillsShSourcePrefix + owner + "/" + repo + "/" + slug,
		Registry: registry, Owner: owner, Repo: repo, Slug: slug,
	}, nil
}

func splitTenantSkillVersion(spec string) (slug, version string) {
	index := strings.LastIndex(spec, "@")
	if index <= 0 {
		return spec, ""
	}
	candidate := spec[index+1:]
	if strings.EqualFold(candidate, "latest") || tenantSkillSemverLike.MatchString(candidate) {
		return spec[:index], candidate
	}
	return spec, ""
}

func splitTenantSkillPath(value string) []string {
	var parts []string
	for _, part := range strings.Split(strings.Trim(value, "/"), "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitTenantSkillOwnerSlug(spec string) (owner, slug string) {
	parts := splitTenantSkillPath(spec)
	switch len(parts) {
	case 0:
		return "", spec
	case 1:
		return "", parts[0]
	case 2:
		return parts[0], parts[1]
	default:
		return "", spec
	}
}

func tenantSkillSlugAndVersion(trimmed, fragment string) (string, string, error) {
	if trimmed == "" {
		return "", "", fmt.Errorf("%w: skill slug is required", ErrSkillSourceInvalid)
	}
	parts := splitTenantSkillPath(trimmed)
	if len(parts) > 0 && strings.EqualFold(parts[0], "skills") {
		parts = parts[1:]
	}
	if len(parts) >= 3 && strings.EqualFold(parts[1], "skills") {
		parts = []string{parts[0], parts[2]}
	}
	if len(parts) > 2 {
		return "", "", fmt.Errorf("%w: unrecognized registry path", ErrSkillSourceInvalid)
	}
	slug := strings.Join(parts, "/")
	version := strings.TrimSpace(fragment)
	slug, fromSpec := splitTenantSkillVersion(slug)
	if fromSpec != "" {
		version = fromSpec
	}
	if slug == "" {
		return "", "", fmt.Errorf("%w: skill slug is required", ErrSkillSourceInvalid)
	}
	return slug, version, nil
}

func (s tenantSkillSource) fetchURL() (string, error) {
	if s.DirectURL != "" {
		return s.DirectURL, nil
	}
	if s.Kind == tenantSkillSourceDirect && s.URL != nil {
		return s.URL.String(), nil
	}
	switch s.Kind {
	case tenantSkillSourceRegistry:
		u, err := url.Parse(strings.TrimRight(s.Registry, "/") + "/api/v1/download")
		if err != nil {
			return "", fmt.Errorf("%w: invalid registry origin", ErrSkillSourceInvalid)
		}
		query := u.Query()
		query.Set("slug", s.Slug)
		if s.Owner != "" {
			query.Set("ownerHandle", s.Owner)
		}
		if s.Version != "" && !strings.EqualFold(s.Version, "latest") {
			if tenantSkillSemverLike.MatchString(s.Version) {
				query.Set("version", s.Version)
			} else {
				query.Set("tag", s.Version)
			}
		}
		u.RawQuery = query.Encode()
		return u.String(), nil
	case tenantSkillSourceSkillsSh:
		u, err := url.Parse(strings.TrimRight(s.Registry, "/") + "/api/v1/skills/" + url.PathEscape(s.Slug) + "/install")
		if err != nil {
			return "", fmt.Errorf("%w: invalid skills.sh registry origin", ErrSkillSourceInvalid)
		}
		query := u.Query()
		query.Set("reference", skillsShSourcePrefix+s.Owner+"/"+s.Repo+"/"+s.Slug)
		u.RawQuery = query.Encode()
		return u.String(), nil
	case tenantSkillSourceGitHub:
		ref := s.GitRef
		if ref == "" {
			ref = "HEAD"
		}
		return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s",
			escapeTenantSkillSegments(s.Owner), url.PathEscape(s.Repo),
			escapeTenantSkillSegments(ref)), nil
	case tenantSkillSourceGitLab:
		ref := s.GitRef
		if ref == "" {
			ref = "HEAD"
		}
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s.zip",
			escapeTenantSkillSegments(s.Owner), url.PathEscape(s.Repo),
			escapeTenantSkillSegments(ref), url.PathEscape(s.Repo+"-"+ref)), nil
	default:
		return "", fmt.Errorf("%w: cannot resolve source", ErrSkillSourceInvalid)
	}
}

func escapeTenantSkillSegments(value string) string {
	parts := strings.Split(value, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func fetchTenantSkillSource(ctx context.Context, source tenantSkillSource, client *http.Client) ([]byte, error) {
	if client == nil {
		client = utils.NewSSRFSafeHTTPClient(utils.SSRFSafeHTTPClientConfig{
			Timeout:      tenantSkillSourceTimeout,
			MaxRedirects: tenantSkillSourceMaxRedirects,
		})
	}
	requestCtx, cancel := context.WithTimeout(ctx, tenantSkillSourceTimeout)
	defer cancel()
	fetched, err := fetchTenantSkillSourceBytes(requestCtx, client, source, 0)
	if err != nil {
		return nil, err
	}
	return normalizeTenantSkillSourcePayloadWithSubdir(fetched.body, fetched.contentType, fetched.subdir)
}

func fetchTenantSkillSourceBytes(
	ctx context.Context,
	client *http.Client,
	source tenantSkillSource,
	hop int,
) (fetchedTenantSkillSource, error) {
	if hop > tenantSkillSourceMaxHops {
		return fetchedTenantSkillSource{}, fmt.Errorf("%w: too many source redirects", ErrSkillSourceInvalid)
	}
	target, err := source.fetchURL()
	if err != nil {
		return fetchedTenantSkillSource{}, err
	}
	body, contentType, err := getTenantSkillSourceURL(ctx, client, target)
	if err != nil {
		return fetchedTenantSkillSource{}, err
	}
	fetched := fetchedTenantSkillSource{
		body: body, contentType: contentType, subdir: source.Subdir, registry: source.Registry,
	}
	if isTenantSkillZIPPayload(contentType, body) || looksLikeTenantSkillMarkdown(body) {
		return fetched, nil
	}
	if !looksLikeTenantSkillJSON(contentType, body) {
		return fetchedTenantSkillSource{}, fmt.Errorf(
			"%w: remote did not return a skill archive", ErrSkillSourceInvalid,
		)
	}
	var handoff tenantSkillSourceHandoff
	if err := json.Unmarshal(body, &handoff); err != nil {
		return fetchedTenantSkillSource{}, fmt.Errorf(
			"%w: remote JSON is not a skill archive", ErrSkillSourceInvalid,
		)
	}
	next, err := tenantSkillSourceFromHandoff(source, handoff)
	if err != nil {
		return fetchedTenantSkillSource{}, err
	}
	return fetchTenantSkillSourceBytes(ctx, client, next, hop+1)
}

func tenantSkillSourceFromHandoff(
	previous tenantSkillSource,
	handoff tenantSkillSourceHandoff,
) (tenantSkillSource, error) {
	if handoff.OK != nil && !*handoff.OK {
		message := strings.TrimSpace(handoff.Message)
		if message == "" {
			message = strings.TrimSpace(handoff.Reason)
		}
		if message == "" {
			message = "registry refused the skill"
		}
		return tenantSkillSource{}, fmt.Errorf("%w: %s", ErrSkillSourceInvalid, truncateTenantSkillError(message))
	}

	archiveURL := strings.TrimSpace(handoff.ArchiveURL)
	if archiveURL == "" {
		archiveURL = strings.TrimSpace(handoff.DownloadURL)
	}
	if archiveURL == "" && handoff.Archive != nil {
		archiveURL = strings.TrimSpace(handoff.Archive.DownloadURL)
	}
	if archiveURL != "" {
		if strings.HasPrefix(archiveURL, "/") {
			if previous.Registry == "" {
				return tenantSkillSource{}, fmt.Errorf("%w: relative archive URL has no registry", ErrSkillSourceInvalid)
			}
			archiveURL = strings.TrimRight(previous.Registry, "/") + archiveURL
		}
		next, err := parseTenantSkillSourceURL(archiveURL)
		if err != nil {
			return tenantSkillSource{}, fmt.Errorf("%w: archive handoff is not usable: %v", ErrSkillSourceInvalid, err)
		}
		next.Subdir = strings.TrimSpace(handoff.Path)
		if next.Subdir == "" && handoff.GitHub != nil {
			next.Subdir = strings.TrimSpace(handoff.GitHub.Path)
		}
		if next.GitRef == "HEAD" && strings.TrimSpace(handoff.Commit) != "" {
			next.GitRef = strings.TrimSpace(handoff.Commit)
		}
		return next, nil
	}

	github := handoff.GitHub
	if github == nil && strings.EqualFold(strings.TrimSpace(handoff.InstallKind), "github") &&
		(strings.TrimSpace(handoff.Repo) != "" || strings.TrimSpace(handoff.Commit) != "") {
		github = &tenantSkillGitHubHandoff{
			Repo: handoff.Repo, Path: handoff.Path, Commit: handoff.Commit,
			SourceURL: handoff.SourceRef,
		}
	}
	if github == nil {
		return tenantSkillSource{}, fmt.Errorf("%w: registry response has no archive URL", ErrSkillSourceInvalid)
	}
	if github.SourceURL != "" {
		next, err := parseTenantSkillSourceURL(github.SourceURL)
		if err == nil && next.Kind == tenantSkillSourceGitHub {
			if github.Commit != "" {
				next.GitRef = github.Commit
			}
			if next.Subdir == "" {
				next.Subdir = strings.Trim(github.Path, "/")
			}
			return next, nil
		}
	}
	owner, repo, ok := splitTenantSkillGitHubRepo(github.Repo)
	if !ok {
		return tenantSkillSource{}, fmt.Errorf("%w: registry GitHub handoff is missing repo", ErrSkillSourceInvalid)
	}
	ref := strings.TrimSpace(github.Commit)
	if ref == "" {
		ref = "HEAD"
	}
	return tenantSkillSource{
		Kind: tenantSkillSourceGitHub, Ref: github.SourceURL,
		Owner: owner, Repo: repo, GitRef: ref,
		Subdir: strings.Trim(github.Path, "/"),
	}, nil
}

func splitTenantSkillGitHubRepo(spec string) (owner, repo string, ok bool) {
	spec = strings.TrimSuffix(strings.TrimSpace(spec), ".git")
	parts := splitTenantSkillPath(spec)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func getTenantSkillSourceURL(ctx context.Context, client *http.Client, rawURL string) ([]byte, string, error) {
	if err := utils.ValidateURLForSSRF(rawURL); err != nil {
		return nil, "", fmt.Errorf("%w: source URL rejected: %v", ErrSkillSourceInvalid, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("%w: invalid source URL", ErrSkillSourceInvalid)
	}
	req.Header.Set("User-Agent", "Ruile-Skill-Catalog/1.0")
	req.Header.Set("Accept", "application/zip, application/octet-stream, application/json, text/markdown, text/plain;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: source download failed: %v", ErrSkillSourceInvalid, err)
	}
	defer resp.Body.Close()
	if resp.ContentLength > maxTenantSkillArchiveBytes {
		return nil, "", fmt.Errorf("%w: source archive is too large", ErrSkillSourceInvalid)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		message := strings.TrimSpace(string(preview))
		if message == "" {
			message = resp.Status
		}
		return nil, "", fmt.Errorf("%w: source returned HTTP %d: %s",
			ErrSkillSourceInvalid, resp.StatusCode, truncateTenantSkillError(message))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTenantSkillArchiveBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: failed to read source response", ErrSkillSourceInvalid)
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("%w: source returned an empty response", ErrSkillSourceInvalid)
	}
	if len(body) > maxTenantSkillArchiveBytes {
		return nil, "", fmt.Errorf("%w: source archive is too large", ErrSkillSourceInvalid)
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func normalizeTenantSkillSourcePayload(data []byte) ([]byte, error) {
	return normalizeTenantSkillSourcePayloadWithSubdir(data, "", "")
}

func normalizeTenantSkillSourcePayloadWithSubdir(data []byte, contentType, subdir string) ([]byte, error) {
	if looksLikeTenantSkillMarkdown(data) {
		var output bytes.Buffer
		writer := zip.NewWriter(&output)
		entry, err := writer.Create(skills.SkillFileName)
		if err != nil {
			return nil, fmt.Errorf("create normalized skill archive: %w", err)
		}
		if _, err := entry.Write(data); err != nil {
			return nil, fmt.Errorf("write normalized skill archive: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close normalized skill archive: %w", err)
		}
		return output.Bytes(), nil
	}
	if !isTenantSkillZIPPayload(contentType, data) {
		return nil, fmt.Errorf("%w: source must return a ZIP archive or SKILL.md", ErrSkillSourceInvalid)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid source ZIP archive: %v", ErrSkillSourceInvalid, err)
	}
	if len(archive.File) == 0 || len(archive.File) > maxTenantSkillArchiveEntries {
		return nil, fmt.Errorf("%w: source archive contains an invalid number of files", ErrSkillSourceInvalid)
	}
	root := strings.Trim(strings.ReplaceAll(subdir, "\\", "/"), "/")
	files := make(map[string][]byte)
	var expandedBytes int64
	for _, entry := range archive.File {
		clean, err := cleanArchivePath(entry.Name)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSkillSourceInvalid, err)
		}
		if entry.FileInfo().Mode()&0o170000 == 0o120000 {
			return nil, fmt.Errorf("%w: source archive cannot contain symlinks", ErrSkillSourceInvalid)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		relative := clean
		if root != "" {
			if clean != root && !strings.HasPrefix(clean, root+"/") {
				continue
			}
			relative = strings.TrimPrefix(strings.TrimPrefix(clean, root), "/")
		}
		if relative == "" {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: open source entry: %v", ErrSkillSourceInvalid, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, maxTenantSkillFileBytes+1))
		_ = reader.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%w: read source entry: %v", ErrSkillSourceInvalid, readErr)
		}
		if len(content) > maxTenantSkillFileBytes {
			return nil, fmt.Errorf("%w: source entry %q is too large", ErrSkillSourceInvalid, relative)
		}
		expandedBytes += int64(len(content))
		if expandedBytes > maxTenantSkillExpandedBytes {
			return nil, fmt.Errorf("%w: source archive expands beyond the allowed limit", ErrSkillSourceInvalid)
		}
		files[relative] = content
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: source subdirectory contains no files", ErrSkillSourceInvalid)
	}
	normalized, err := buildTenantSkillArchive(files)
	if err != nil {
		return nil, fmt.Errorf("%w: build normalized source archive: %v", ErrSkillSourceInvalid, err)
	}
	if _, _, err := inspectTenantSkillArchive(normalized); err != nil {
		return nil, fmt.Errorf("%w: source archive does not contain exactly one valid SKILL.md: %v", ErrSkillSourceInvalid, err)
	}
	return normalized, nil
}

func buildTenantSkillArchive(files map[string][]byte) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entry.Write(content); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func isGitHubArchivePath(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "/archive/") && strings.HasSuffix(lower, ".zip")
}

func isGitLabArchivePath(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "/-/archive/") && strings.HasSuffix(lower, ".zip")
}

func isDirectSkillPath(value string) bool {
	lower := strings.ToLower(value)
	for _, suffix := range []string{".zip", ".md"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func isTenantSkillZIPPayload(contentType string, body []byte) bool {
	if len(body) >= 4 && (bytes.HasPrefix(body, []byte("PK\x03\x04")) ||
		bytes.HasPrefix(body, []byte("PK\x05\x06")) ||
		bytes.HasPrefix(body, []byte("PK\x07\x08"))) {
		return true
	}
	return strings.Contains(strings.ToLower(contentType), "zip")
}

func looksLikeTenantSkillJSON(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func looksLikeTenantSkillMarkdown(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return false
	}
	return bytes.Contains(trimmed, []byte("\nname:")) ||
		bytes.Contains(trimmed, []byte("\nname :"))
}

func hasSensitiveSourceQuery(query url.Values) bool {
	for key := range query {
		lower := strings.ToLower(key)
		for _, marker := range []string{
			"token", "secret", "password", "passwd", "api_key", "apikey",
			"access_key", "authorization",
		} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func isClawHubOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	return err == nil && isClawHubHost(parsed.Hostname())
}

func truncateTenantSkillError(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 200 {
		return value[:200]
	}
	return value
}
