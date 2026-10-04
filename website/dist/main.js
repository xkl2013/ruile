// 睿乐大脑官网 v2 — 交互脚本

// 预约演示表单：填入飞书表单链接后，全站「预约演示」入口自动指向该表单（新开页）。
// 留空 = 维持现状（跳转首页 #contact 区块）。
const DEMO_FORM_URL = "";

if (DEMO_FORM_URL) {
  document.querySelectorAll('a[href$="#contact"]').forEach((a) => {
    if ((a.textContent || "").includes("预约演示")) {
      a.setAttribute("href", DEMO_FORM_URL);
      a.setAttribute("target", "_blank");
      a.setAttribute("rel", "noreferrer");
    }
  });
}

// 顶部导航：滚动阴影
const header = document.querySelector(".site-header");
if (header) {
  const onScroll = () => header.classList.toggle("is-scrolled", window.scrollY > 8);
  window.addEventListener("scroll", onScroll, { passive: true });
  onScroll();
}

// 移动端菜单
const menuToggle = document.querySelector(".menu-toggle");
const mobileNav = document.querySelector(".mobile-nav");
if (header && menuToggle && mobileNav) {
  menuToggle.addEventListener("click", () => {
    const isOpen = header.classList.toggle("menu-open");
    menuToggle.setAttribute("aria-expanded", String(isOpen));
    mobileNav.setAttribute("aria-hidden", String(!isOpen));
  });
  mobileNav.querySelectorAll("a").forEach((link) => {
    link.addEventListener("click", () => {
      header.classList.remove("menu-open");
      menuToggle.setAttribute("aria-expanded", "false");
      mobileNav.setAttribute("aria-hidden", "true");
    });
  });
}

// 当前页导航高亮（仅精确匹配页面文件，锚点链接不参与）
const here = location.pathname.split("/").pop() || "index.html";
document.querySelectorAll(".desktop-nav a, .mobile-nav a").forEach((a) => {
  const target = a.getAttribute("href") || "";
  if (target === here) a.classList.add("is-current");
});

// 能力 / 模块 Tab 切换
const capabilityTabs = document.querySelectorAll(".capability-tab");
const capabilityPanels = document.querySelectorAll(".capability-panel");
capabilityTabs.forEach((tab) => {
  tab.addEventListener("click", () => {
    const target = tab.dataset.tab;
    if (!target) return;
    capabilityTabs.forEach((item) => {
      const active = item === tab;
      item.classList.toggle("is-active", active);
      item.setAttribute("aria-selected", String(active));
    });
    capabilityPanels.forEach((panel) => {
      panel.classList.toggle("is-active", panel.dataset.panel === target);
    });
  });
});

// FAQ 手风琴符号
document.querySelectorAll(".faq-list details").forEach((item) => {
  item.addEventListener("toggle", () => {
    const symbol = item.querySelector("summary b");
    if (symbol) symbol.textContent = item.open ? "−" : "＋";
  });
});

// 滚动进入动画
const revealItems = document.querySelectorAll(".reveal");
if ("IntersectionObserver" in window) {
  const revealObserver = new IntersectionObserver(
    (entries, observer) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return;
        entry.target.classList.add("is-visible");
        observer.unobserve(entry.target);
      });
    },
    { threshold: 0.12 }
  );
  revealItems.forEach((item) => revealObserver.observe(item));
} else {
  revealItems.forEach((item) => item.classList.add("is-visible"));
}
