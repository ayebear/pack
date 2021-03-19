#!/usr/bin/env node
const path = require("path");
const { spawn } = require("child_process");
spawn(path.join(__dirname, "./pack"), [process.argv.slice(2)], {
  stdio: "inherit",
});
