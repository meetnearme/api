import tailwindcss from 'tailwindcss';
import crypto from 'crypto';
import fs from 'fs';

let previousHash = '';

export default {
  plugins: [
    tailwindcss,
    {
      postcssPlugin: 'css-watch-logger',
      Once(root, { result }) {
        try {
          // Generate hash from CSS content
          const css = root.toString();
          const newHash = crypto.createHash('md5').update(css).digest('hex').slice(0, 8);

          // Read the template file
          const templatePath = 'functions/gateway/templates/pages/layout.templ';
          const template = fs.readFileSync(templatePath, 'utf8');

          // Extract current hash if it exists
          const hashMatch = template.match(/___([^_]+)___/);
          const currentHash = hashMatch ? hashMatch[1] : null;

          // Only update if both conditions are met:
          // 1. New hash is different from the current hash in the file
          // 2. New hash is different from the previous hash we processed
          if (newHash !== currentHash && newHash !== previousHash) {
            const pattern = /___[^_]+___/;
            const updatedTemplate = template.replace(pattern, `___${newHash}___`);
            fs.writeFileSync(templatePath, updatedTemplate);
            console.log(`🔄 CSS hash changed, updated in layout.templ: ${newHash}`);
          }

          // Update previousHash for next run
          previousHash = newHash;
        } catch (error) {
          console.warn('Warning: Could not update CSS hash:', error.message);
        }
      }
    }
  ]
};
