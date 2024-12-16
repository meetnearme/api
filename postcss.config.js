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

          // Update pattern to match the old style with query parameter
          const pattern = /styles\.css\?h=___[^_]+___/;
          const updatedTemplate = template.replace(pattern, `styles.${newHash}.css`);

          if (template !== updatedTemplate) {
            fs.writeFileSync(templatePath, updatedTemplate);

            // Wait for the temp file to be written
            setTimeout(() => {
              if (fs.existsSync(tempFile)) {
                // Copy temp file to hashed filename
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
            }, 100); // Small delay to ensure file is written
          }

          previousHash = newHash;
        } catch (error) {
          console.warn('Warning: Could not update CSS:', error.message);
        }
      }
    }
  ]
};
