import tailwindcss from 'tailwindcss';
import { readFile, writeFile, unlink } from 'fs/promises';
import { readdirSync } from 'fs';
import crypto from 'crypto';
import process from 'node:process';

// Helper function to generate unique hash for filename
function generateHash(content) {
  return crypto.createHash('md5').update(content).digest('hex').slice(0, 8);
}

async function updateCSSFiles(finalCSS, result) {
  const outputPath = result.opts.to;
  if (!outputPath) return;

  // Check if we're in production mode
  const isProduction = process.env.NODE_ENV === 'production';

  if (!isProduction) {
    // In dev mode, just write the CSS file without hashing
    console.log(`📦 CSS file created: styles.css (dev mode)`);
    return;
  }

  // Production mode: apply hashing
  const hash = generateHash(finalCSS);
  const hashedFilename = `styles.${hash}.css`;
  const hashedPath = outputPath.replace(/styles\.css$/, hashedFilename);

  try {
    // Write the new hashed file
    await writeFile(hashedPath, finalCSS);
    console.log(`📦 CSS file created: ${hashedFilename}`);

    // Read current layout.templ content
    const layoutPath = './functions/gateway/templates/pages/layout.templ';
    const layoutContent = await readFile(layoutPath, 'utf8');

    // Update the CSS filename reference
    const updatedContent = layoutContent.replace(
      /styles\.[a-f0-9]{8}\.css/g,
      hashedFilename,
    );

    if (updatedContent !== layoutContent) {
      await writeFile(layoutPath, updatedContent);
      console.log(
        `✅ Updated layout.templ with new CSS filename: ${hashedFilename}`,
      );
    }
  } catch (error) {
    console.warn(
      'Warning: Could not update CSS filename in layout.templ:',
      error.message,
    );
  }
}

export default {
  plugins: [
    tailwindcss,
    {
      postcssPlugin: 'css-watch-logger',
      Once(root, { result }) {
        // Only update CSS files in production mode
        if (process.env.NODE_ENV === 'production') {
          updateCSSFiles(root.toString(), result).catch(console.error);
        } else {
          console.log(`📦 CSS file updated: styles.css (dev mode)`);
        }
      },
    },
  ],
};
