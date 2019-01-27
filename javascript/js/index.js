require("babel-core/register");
require("babel-polyfill");

import Tupelo from './src/tupelo';

let tupelo = new Tupelo(global.V8Worker2);

export default tupelo;
