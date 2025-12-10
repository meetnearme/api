var config = {
    keywords: window.broadstreetKeywords || [],
    softKeywords: true
};

///////////////////////////////////////////////
// START - PASSING OF URL AS A KEYWORDS
// https://app.asana.com/0/1185128397849241/1209518767354453
///////////////////////////////////////////////

const { pathname } = location;
config.keywords.push(...(pathname !== '/' ? pathname.split('/').filter(Boolean) : ['home']));
window.broadstreetKeywords = window.broadstreetKeywords || config.keywords;

///////////////////////////////////////////////
// END - PASSING OF URL AS A KEYWORDS
///////////////////////////////////////////////

broadstreet.addCSS(`broadstreet-zone > div > span {display: block;}`);

broadstreet.watch(config);