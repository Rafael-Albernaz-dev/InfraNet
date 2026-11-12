import { execFileSync } from "node:child_process";

const paths = execFileSync("git", ["ls-files", "-z", "--", "internal/"], {
  encoding: "utf8",
})
  .split("\0")
  .filter(Boolean);

for (const path of paths) {
  if (!path.endsWith(".go") && !/\/(testdata|migrations)\//.test(path)) {
    console.error(`${path}: move non-Go file into testdata/ or migrations/`);
    process.exitCode = 1;
  }
}
