import tailwindcss from 'tailwindcss';
import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

let previousHash = '';
let previousTemplateHash = '';

export default {
  plugins: [
    tailwindcss,
    {
      postcssPlugin: 'css-watch-logger',
      Once(root, { result }) {
        try {
          const css = root.toString();
          const resultString = JSON.stringify(result.messages);
          const combinedContent = css + resultString;
          const newHash = crypto.createHash('md5').update(combinedContent).digest('hex').slice(0, 8);

          // Define paths
          const baseStylesPath = './static/assets/styles';
          const tempFile = `${baseStylesPath}.css`;
          const newFileName = `${baseStylesPath}.${newHash}.css`;

          // Read the template file
          const templatePath = 'functions/gateway/templates/pages/layout.templ';
          const template = fs.readFileSync(templatePath, 'utf8');

          // Hash the template content to detect changes
          const templateHash = crypto.createHash('md5').update(template).digest('hex').slice(0, 8);

          // Extract current hash from filename in template
          const hashMatch = template.match(/styles\.(.*?)\.css/);
          const currentHash = hashMatch ? hashMatch[1] : null;

          // Check if we need to update based on:
          // 1. Hash differences
          // 2. Missing hashed CSS file
          // 3. Template content hasn't changed
          const currentHashedFile = currentHash ? `${baseStylesPath}.${currentHash}.css` : null;
          const needsUpdate = (newHash !== currentHash && newHash !== previousHash ||
                             (currentHash && !fs.existsSync(currentHashedFile))) &&
                             templateHash !== previousTemplateHash;

          if (needsUpdate) {
            previousTemplateHash = templateHash;  // Store the new template hash
            const pattern = /styles\..*?\.css/;
            const updatedTemplate = template.replace(pattern, `styles.${newHash}.css`);
            fs.writeFileSync(templatePath, updatedTemplate);

            // Ensure the temp file exists and copy it
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
            } else {
              console.warn('Warning: styles.css not found for initial copy');
            }
          }

          previousHash = newHash;
        } catch (error) {
          console.warn('Warning: Could not update CSS:', error.message);
        }
      }
    }
  ]
};
