import watch from 'node-watch';
import { spawn } from 'child_process';
import path from 'path';

watch(process.argv[2], { recursive: true }, (eventType, filename) => {
  const fileExtension = path.extname(filename);
  if (fileExtension === '.templ') {
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
      console.error(`Error running 'templ generate' command: ${error.message}`);
    });

    childProcess.on('close', (code) => {
      if (code === 0) {
        console.log(`'templ generate' command executed successfully.`);
      } else {
        console.error(`'templ generate' command exited with code ${code}.`);
      }
    });
  }
});
