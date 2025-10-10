import watch from 'node-watch';
import { spawn } from 'child_process';
import path from 'path';
import fs from 'fs';
import process from 'node:process';
const dirs = process.argv.slice(2);

const layoutTemplPath = 'functions/gateway/templates/pages/layout.templ';
let prevLayoutTempl = '';

// Function to read the current content of layout.templ
function readLayoutTempl() {
  return fs.readFileSync(layoutTemplPath, 'utf8');
}

dirs.forEach((dir) => {
  watch(
    dir,
    {
      recursive: true,
    },
    (eventType, filename) => {
      const fileExtension = path.extname(filename);
      if (fileExtension === '.templ') {
        if (filename === layoutTemplPath) {
          const currentContent = readLayoutTempl();

          // Check if the content has changed
          if (currentContent === prevLayoutTempl) {
            console.log(
              `No changes detected in layout.templ. Skipping 'templ generate' command.`,
            );
            return;
          }

          prevLayoutTempl = currentContent;
        }

        console.log(
          `File ${filename} has been ${eventType}d. Running 'templ generate' command...`,
        );
        const childProcess = spawn('templ', ['generate']);

        childProcess.stdout.on('data', (data) => {
          process.stdout.write(data);
        });

        childProcess.stderr.on('data', (data) => {
          process.stderr.write(data);
        });

        childProcess.on('error', (error) => {
          console.error(
            `Error running 'templ generate' command: ${error.message}`,
          );
        });

        childProcess.on('close', (code) => {
          if (code === 0) {
            console.log(`'templ generate' command executed successfully.`);
          } else {
            console.error(`'templ generate' command exited with code ${code}.`);
          }
        });
      }
    },
  );
});
