import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import process from 'node:process';

const version = process.argv[2] || process.env.VERSION;

if (!version) {
  console.error('Usage: node scripts/sync-client-version.mjs <version>');
  process.exit(1);
}

if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
  console.error(`Invalid version: ${version}`);
  process.exit(1);
}

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDir, '..');
const files = {
  tauri: path.join(projectRoot, 'client', 'src-tauri', 'tauri.conf.json'),
  cargo: path.join(projectRoot, 'client', 'src-tauri', 'Cargo.toml'),
  cargoLock: path.join(projectRoot, 'client', 'src-tauri', 'Cargo.lock'),
  packageJson: path.join(projectRoot, 'client', 'package.json'),
  packageLock: path.join(projectRoot, 'client', 'package-lock.json'),
};

function writeJsonVersion(filePath) {
  const original = fs.readFileSync(filePath, 'utf8');
  const versionPattern = /("version"\s*:\s*")([^"]+)(")/;
  const match = original.match(versionPattern);

  JSON.parse(original);

  if (!match) {
    throw new Error(`Failed to locate version in ${path.relative(projectRoot, filePath)}`);
  }

  const previousVersion = match[2];
  const updated = original.replace(versionPattern, `$1${version}$3`);

  fs.writeFileSync(filePath, updated, 'utf8');

  return previousVersion;
}

function writeCargoVersion(filePath) {
  const original = fs.readFileSync(filePath, 'utf8');
  const packageVersionPattern = /(^\[package\]\r?\n[\s\S]*?^version\s*=\s*")([^"]+)(")/m;
  const match = original.match(packageVersionPattern);

  if (!match) {
    throw new Error(`Failed to locate package version in ${path.relative(projectRoot, filePath)}`);
  }

  const previousVersion = match[2];
  const updated = original.replace(packageVersionPattern, `$1${version}$3`);

  fs.writeFileSync(filePath, updated, 'utf8');

  return previousVersion;
}

function writeCargoLockVersion(filePath) {
  const original = fs.readFileSync(filePath, 'utf8');
  const packageVersionPattern = /(\[\[package\]\]\r?\nname = "rpbox"\r?\nversion = ")([^"]+)(")/;
  const match = original.match(packageVersionPattern);

  if (!match) {
    throw new Error(`Failed to locate rpbox package version in ${path.relative(projectRoot, filePath)}`);
  }

  fs.writeFileSync(filePath, original.replace(packageVersionPattern, `$1${version}$3`), 'utf8');
  return match[2];
}

function writePackageLockVersion(filePath) {
  const original = fs.readFileSync(filePath, 'utf8');
  JSON.parse(original);
  const packageVersionPattern = /("name": "rpbox-client",\r?\n\s*"version": ")([^"]+)(")/g;
  const matches = [...original.matchAll(packageVersionPattern)];

  if (matches.length !== 2) {
    throw new Error(`Expected two rpbox-client versions in ${path.relative(projectRoot, filePath)}, found ${matches.length}`);
  }

  fs.writeFileSync(filePath, original.replace(packageVersionPattern, `$1${version}$3`), 'utf8');
  return matches[0][2];
}

try {
  const tauriPrevious = writeJsonVersion(files.tauri);
  const cargoPrevious = writeCargoVersion(files.cargo);
  const cargoLockPrevious = writeCargoLockVersion(files.cargoLock);
  const packagePrevious = writeJsonVersion(files.packageJson);
  const packageLockPrevious = writePackageLockVersion(files.packageLock);

  console.log(`Updated client/src-tauri/tauri.conf.json: ${tauriPrevious} -> ${version}`);
  console.log(`Updated client/src-tauri/Cargo.toml: ${cargoPrevious} -> ${version}`);
  console.log(`Updated client/src-tauri/Cargo.lock: ${cargoLockPrevious} -> ${version}`);
  console.log(`Updated client/package.json: ${packagePrevious} -> ${version}`);
  console.log(`Updated client/package-lock.json: ${packageLockPrevious} -> ${version}`);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
