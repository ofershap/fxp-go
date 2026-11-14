// Generates differential goldens from fast-xml-parser (pinned version).
// Each case: {name, xml, options}. Output: goldens.json [{name, out?, error?}].
const fs = require('fs');
const { XMLParser, XMLValidator } = require('fast-xml-parser');

const cases = JSON.parse(fs.readFileSync(__dirname + '/cases.json', 'utf8'));
const goldens = cases.map((c) => {
  try {
    const parser = new XMLParser(c.options);
    let out = parser.parse(c.xml);
    return { name: c.name, out };
  } catch (e) {
    return { name: c.name, error: String(e && e.message ? e.message : e) };
  }
});
fs.writeFileSync(__dirname + '/goldens.json', JSON.stringify(goldens, null, 1));
console.log('wrote', goldens.length, 'goldens,', goldens.filter(g => g.error).length, 'errors');
console.log('fxp version:', JSON.parse(require('fs').readFileSync(__dirname+'/node_modules/fast-xml-parser/package.json')).version);
