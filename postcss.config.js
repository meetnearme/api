import tailwindcss from 'tailwindcss';
import crypto from 'crypto';
import fs from 'fs';

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

          // Extract current hash if it exists (now including the ___ wrapping)
          const hashMatch = template.match(/___([^_]+)___/);
          const currentHash = hashMatch ? hashMatch[1] : null;
          if (newHash !== currentHash) {
            const pattern = /___[^_]+___/;  // Match content between ___ excluding underscores
            const updatedTemplate = template.replace(pattern, `___${newHash}___`);
            // Only write if content actually changed
            if (updatedTemplate !== template) {
              fs.writeFileSync(templatePath, updatedTemplate);
              console.log(`🔄 CSS hash changed, updated in layout.templ: ${newHash}`);
            }
          }
        } catch (error) {
          console.warn('Warning: Could not update CSS hash:', error.message);
        }
      }
    }
  ]
};
