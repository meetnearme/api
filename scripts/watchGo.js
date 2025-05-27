
import watch from 'node-watch';
import { spawn, exec } from 'child_process';
import path from 'path';
import fs from 'fs';
const dirs = process.argv.slice(2);

const WATCH_DIRECTORIES = ['./functions']; // Adjust as needed
// Main Go package to build
const GO_MAIN_PACKAGE = './functions/gateway/main.go'; // Adjust if needed
// Output path for the binary ON THE HOST (maps to /go-app/main in container)
const OUTPUT_BINARY_PATH = './docker_build/main';

const CONTAINER_NAME = "meetnearme-go-app"; // <<< MUST match the --name flag in your docker:dev:run script
const SUPERVISOR_PROGRAM_NAME = "go-app"; // <<< MUST match the [program:...] name in supervisord.dev.conf

dirs.forEach((dir) => {
  watch(
    dir,
    {
      recursive: true,
    },
    (eventType, filename) => {
      const fileExtension = path.extname(filename);
      if (fileExtension === '.go') {
        const args = [
          'build',
          '-o', OUTPUT_BINARY_PATH, // Output path RELATIVE TO HOST SCRIPT
          GO_MAIN_PACKAGE // Path to main package RELATIVE TO HOST SCRIPT
        ];

        // Environment variables for cross-compilation
        const buildEnv = {
          ...process.env, // Inherit existing host environment variables
          CGO_ENABLED: '0', // Disable CGO
          GOOS: 'linux', // Target Linux OS
          // GOARCH: 'amd64' // Explicitly set architecture if needed, often inferred
        };

        // Spawn the process
        const childProcess = spawn('go', args, {
          env: buildEnv     // Pass the modified environment variables
        });

        console.log(`Running restart command for Go App in Docker`);

        childProcess.stdout.on('data', (data) => {
          process.stdout.write(data);
        });

        childProcess.stderr.on('data', (data) => {
          process.stderr.write(data);
        });

        childProcess.on('error', (error) => {
          console.error(
            `Error running 'go build' command: ${error.message}`,
          );
        });

        childProcess.on('close', (code) => {
          if (code === 0) {
            console.log(`'go build' command executed successfully.`);
            triggerContainerRestart();
          } else {
            console.error(`'go build' command exited with code ${code}.`);
          }
        });
      }
    },
  );
});


function triggerContainerRestart() {
  console.log(`[Restart Trigger] Checking if container '${CONTAINER_NAME}' is running...`);
  const checkArgs = [
    'ps',
    '-q',
    '-f',
    `name=^/${CONTAINER_NAME}$`
  ]

  const checkCommandChildProcess = spawn('docker', checkArgs);

  let containerId = '';
  let checkStderr = '';

  checkCommandChildProcess.stdout.on('data', (data) => {
    containerId += data.toString().trim()
  })

  checkCommandChildProcess.stderr.on('data', (data) => {
    console.error(`[Docker Check STDERR] ${data.toString()}`);
  })

  checkCommandChildProcess.on('error', (error) => {
    console.error(`[Docker Check SPAWN ERROR] Failed to start:  ${error.message}`);
  })

  checkCommandChildProcess.on('close', (code) => {
    if (code !== 0) {
      console.error(`[Docker Check] docker ps command exited with code:  ${code}`);
      return
    }

    if (containerId) {
      console.log(`[Docker Check] Container '${CONTAINER_NAME}' is running (ID: ${containerId}).`);

      const restartArgs = [
        'exec',
        CONTAINER_NAME,
        'supervisorctl',
        '-s',
        'unix:///run/supervisor/supervisor.sock',
        'restart',
        SUPERVISOR_PROGRAM_NAME
      ];

      console.log(`[Debug] Running: docker ${restartArgs.join(' ')}`);

      const restartProcess = spawn('docker', restartArgs)

      let restartStdout = ''; // To capture output from supervisorctl
      let restartStderr = ''; // To capture errors from supervisorctl or docker exec


      restartProcess.stdout.on('data', (data) => {
        restartStdout += data.toString();
      });

      restartProcess.stderr.on('data', (data) => {
        restartStderr += data.toString();
      });

      restartProcess.on('error', (error) => {
        // Handle errors starting the 'docker exec' process
        console.error(`[Docker Restart SPAWN ERROR] Failed to start 'docker exec': ${error.message}`);
      });

      restartProcess.on('close', (restartCode) => {
        if (restartCode === 0) {
          console.log(`[Restart Trigger] 'docker exec supervisorctl restart' command completed successfully.`);
          // Log output which might confirm restart
          if (restartStdout) console.log(`[Restart Trigger] stdout: ${restartStdout.trim()}`);
          // supervisorctl often outputs success/status to stderr
          if (restartStderr) console.log(`[Restart Trigger] stderr: ${restartStderr.trim()}`);
        } else {
          console.error(`[Restart Trigger] 'docker exec supervisorctl restart' failed with code ${restartCode}.`);
          if (restartStdout) console.error(`[Restart Trigger] stdout: ${restartStdout.trim()}`);
          if (restartStderr) console.error(`[Restart Trigger] stderr: ${restartStderr.trim()}`);
        }
      })
    } else {
      console.log(`[Docker Check] Container '${CONTAINER_NAME}' is not running or not found.`);
      if (checkStderr) console.warn(`[Docker Check] stderr from 'docker ps': ${checkStderr.trim()}`);
    }
  })
}

