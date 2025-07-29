import tailwindcss from 'tailwindcss';
import crypto from 'crypto';
import fs from 'fs';
import path from 'path';

function updateCSSFiles(css, result) {
  // Define paths
  const baseStylesPath = './static/assets/styles';
  const templatePath = 'functions/gateway/templates/pages/layout.templ';

  // if (process.env.NODE_ENV === 'production') {
  if (process.env.SST_STAGE === 'prod') {
    const resultString = JSON.stringify(result.messages);
    const combinedContent = css + resultString;
    const newHash = crypto.createHash('md5').update(combinedContent).digest('hex').slice(0, 8);

    const newFileName = `${baseStylesPath}.${newHash}.css`;
    fs.writeFileSync(tempFile, css);

    // Update template with new hash
    if (fs.existsSync(templatePath)) {
      const template = fs.readFileSync(templatePath, 'utf8');
      const pattern = /styles\..*?\.css/;
      const updatedTemplate = template.replace(pattern, `styles.${newHash}.css`);
      fs.writeFileSync(templatePath, updatedTemplate);
    }

    console.log(`Production CSS generated: ${newFileName}`);
    return;
  } else {
    const devFile = `${baseStylesPath}.css`;
    fs.writeFileSync(devFile, css);
    console.log(`Development CSS updated: ${devFile}`);
    return;
  }
}

export default {
  plugins: [
    tailwindcss,
    {
      postcssPlugin: 'css-watch-logger',
      Once(root, { result }) {
        updateCSSFiles(root.toString(), result);
      }
    }
  ]
};
