#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readdirSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const web = join(root, "web");
const isWindows = process.platform === "win32";

const goCommand = process.env.PIM_GO || selectGoCommand();
const gofmtCommand = process.env.PIM_GOFMT || selectGofmtCommand(goCommand);
const goPackages = ["./cmd/...", "./internal/..."];

function selectGoCommand() {
  const portableGo = join(root, ".tools", "go", "bin", "go.exe");
  return isWindows && existsSync(portableGo) ? portableGo : "go";
}

function selectGofmtCommand(selectedGo) {
  if (selectedGo !== "go") {
    return join(dirname(selectedGo), isWindows ? "gofmt.exe" : "gofmt");
  }

  return "gofmt";
}

function displayPath(value) {
  const local = relative(root, value);
  return local && !local.startsWith("..") ? local : value;
}

function displayCommand(command, args) {
  return [displayPath(command), ...args].join(" ");
}

function run(command, args, options = {}) {
  console.log(`> ${displayCommand(command, args)}`);

  const result = spawnSync(command, args, {
    cwd: options.cwd || root,
    env: process.env,
    stdio: "inherit",
    shell: options.shell || false,
  });

  if (result.error) {
    console.error(
      `Unable to start ${displayPath(command)}: ${result.error.message}`,
    );
    process.exit(1);
  }

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function runNpm(args, options = {}) {
  if (process.env.PIM_NPM) {
    run(process.env.PIM_NPM, args, options);
    return;
  }

  if (process.env.npm_execpath && existsSync(process.env.npm_execpath)) {
    run(process.execPath, [process.env.npm_execpath, ...args], options);
    return;
  }

  run(isWindows ? "npm.cmd" : "npm", args, {
    ...options,
    shell: isWindows,
  });
}

function runCaptured(command, args) {
  console.log(`> ${displayCommand(command, args)}`);

  const result = spawnSync(command, args, {
    cwd: root,
    env: process.env,
    encoding: "utf8",
    shell: false,
  });

  if (result.error) {
    console.error(
      `Unable to start ${displayPath(command)}: ${result.error.message}`,
    );
    process.exit(1);
  }

  if (result.stderr) {
    process.stderr.write(result.stderr);
  }

  if (result.status !== 0) {
    if (result.stdout) {
      process.stdout.write(result.stdout);
    }
    process.exit(result.status ?? 1);
  }

  return result.stdout.trim();
}

function collectGoFiles(directory) {
  if (!existsSync(directory)) {
    return [];
  }

  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const entryPath = join(directory, entry.name);
      if (entry.isDirectory()) {
        return collectGoFiles(entryPath);
      }
      return entry.isFile() && entry.name.endsWith(".go") ? [entryPath] : [];
    })
    .sort();
}

function allGoFiles() {
  return ["cmd", "internal", "tests"].flatMap((directory) =>
    collectGoFiles(join(root, directory)),
  );
}

function format() {
  const files = allGoFiles();
  if (files.length > 0) {
    run(gofmtCommand, ["-w", ...files]);
  }
  runNpm(["run", "format"], { cwd: web });
}

function formatCheck() {
  const files = allGoFiles();
  if (files.length > 0) {
    const unformatted = runCaptured(gofmtCommand, ["-l", ...files]);
    if (unformatted) {
      console.error("Go files need formatting:");
      console.error(unformatted);
      process.exit(1);
    }
  }
  runNpm(["run", "format:check"], { cwd: web });
}

function lint() {
  run(goCommand, ["vet", ...goPackages]);
  runNpm(["run", "typecheck"], { cwd: web });
}

function test() {
  run(goCommand, ["test", ...goPackages]);
  runNpm(["run", "test"], { cwd: web });
}

function build() {
  const outputDirectory = join(root, ".tools", "build");
  mkdirSync(outputDirectory, { recursive: true });
  const binary = join(outputDirectory, isWindows ? "pim.exe" : "pim");

  run(goCommand, ["build", "-trimpath", "-o", binary, "./cmd/pim"]);
  runNpm(["run", "build"], { cwd: web });
}

function reviewDependencies() {
  run(goCommand, ["mod", "verify"]);
  run(goCommand, ["list", "-m", "all"]);
  runNpm(["ls", "--depth=0"], { cwd: web });
  runNpm(["audit", "--audit-level=high"], { cwd: web });
}

function check() {
  formatCheck();
  lint();
  test();
}

function checkFull() {
  check();
  build();
  reviewDependencies();
}

function help() {
  console.log(`Infrastructure Monitor quality commands

  npm run format         Format Go and frontend source files
  npm run format:check   Verify formatting without changing files
  npm run lint           Run Go vet and TypeScript static checks
  npm test               Run Go unit tests and frontend tests
  npm run build          Build the Go executable and frontend assets
  npm run deps:review    Verify and audit locked dependencies
  npm run check          Run the fast, local, non-production check
  npm run check:full     Run the fast check, builds, and dependency review

The fast check does not require Docker, production secrets, or network access.`);
}

const commands = new Map([
  ["help", help],
  ["format", format],
  ["format-check", formatCheck],
  ["lint", lint],
  ["test", test],
  ["build", build],
  ["deps-review", reviewDependencies],
  ["check", check],
  ["check-full", checkFull],
]);

const requestedCommand = process.argv[2] || "help";
const command = commands.get(requestedCommand);

if (!command) {
  console.error(`Unknown quality command: ${requestedCommand}`);
  help();
  process.exit(2);
}

command();
