import tailwindcss from 'tailwindcss';
import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

let previousHash = '';

export default {
  plugins: [
    tailwindcss,
    {
      postcssPlugin: 'css-watch-logger',
      Once(root, { result }) {
        try {
          const css = root.toString();
          const newHash = crypto.createHash('md5').update(css).digest('hex').slice(0, 8);
          // Define paths
          const baseStylesPath = './static/assets/styles';
          const tempFile = `${baseStylesPath}.css`;
          const newFileName = `${baseStylesPath}.${newHash}.css`;

          // Read the template file
          const templatePath = 'functions/gateway/templates/pages/layout.templ';
          const template = fs.readFileSync(templatePath, 'utf8');

          // Extract current hash from filename in template
          const hashMatch = template.match(/styles\.(.*?)\.css/);
          const currentHash = hashMatch ? hashMatch[1] : null;

          // Only update if both conditions are met:
          // 1. New hash is different from the current hash in the file
          // 2. New hash is different from the previous hash we processed
          if (newHash !== currentHash && newHash !== previousHash) {
            const pattern = /styles\..*?\.css/;
            const updatedTemplate = template.replace(pattern, `styles.${newHash}.css`);
            fs.writeFileSync(templatePath, updatedTemplate);

            // Wait for the temp file to be written
            setTimeout(() => {
              if (fs.existsSync(tempFile)) {
                fs.copyFileSync(tempFile, newFileName);

                // Remove old CSS file if it exists
                if (previousHash) {
                  const oldFile = `${baseStylesPath}.${previousHash}.css`;
                  if (fs.existsSync(oldFile)) {
                    fs.unlinkSync(oldFile);
                  }
                }

                console.log(`🔄 CSS updated: ${newFileName}`);
              }
            }, 100);
          }

          previousHash = newHash;
        } catch (error) {
          console.warn('Warning: Could not update CSS:', error.message);
        }
      }
    }
  ]
};
