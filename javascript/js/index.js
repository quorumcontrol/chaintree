const Tupelo = require('./src/tupelo');

let tupelo = new Tupelo(global.V8Worker2);

module.exports = tupelo;
