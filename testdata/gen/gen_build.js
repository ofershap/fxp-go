// Builds XML from each parse golden with the JS XMLBuilder, for
// semantic roundtrip comparison (both sides re-parsed and deep-compared;
// Go map output is sorted-key deterministic, JS preserves insertion order).
const fs = require('fs');
const { XMLBuilder } = require('fast-xml-parser');
const cases = JSON.parse(fs.readFileSync(__dirname + '/cases.json', 'utf8'));
const goldens = JSON.parse(fs.readFileSync(__dirname + '/goldens.json', 'utf8'));
const gmap = Object.fromEntries(goldens.map((g) => [g.name, g]));
const out = [];
for (const c of cases) {
  const g = gmap[c.name];
  if (!g || g.error) { out.push({ name: c.name, skip: 'parse-error' }); continue; }
  if (c.options && (c.options.preserveOrder || c.options.stopNodes)) { out.push({ name: c.name, skip: 'ordered/stop' }); continue; }
  try {
    const bopts = {};
    if (c.options) {
      for (const k of ['ignoreAttributes','attributeNamePrefix','attributesGroupName','textNodeName','cdataPropName','commentPropName','processEntities']) {
        if (k in c.options) bopts[k] = c.options[k];
      }
      bopts.ignoreAttributes = !!(c.options.ignoreAttributes);
    }
    const xml = new XMLBuilder(bopts).build(g.out);
    out.push({ name: c.name, xml });
  } catch (e) {
    out.push({ name: c.name, error: String(e && e.message ? e.message : e) });
  }
}
fs.writeFileSync(__dirname + '/build_goldens.json', JSON.stringify(out, null, 1));
console.log('wrote', out.length, 'build goldens');
