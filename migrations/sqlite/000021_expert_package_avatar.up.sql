-- Persist the display avatar extracted from imported expert packages.
ALTER TABLE expert_packages ADD COLUMN avatar TEXT NOT NULL DEFAULT '';
