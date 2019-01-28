import Tupelo from './src/tupelo.mjs';

global.onerror = function(message, source, lineno, colno, error) {
    V8Worker2.print("msg: ", message, " source: ", source, " lineno: ", lineno, " err: ", error); 
    return true;
}

global.setImmediate = (fn) => {
    fn();
}

global.clearImmediate = (imm) => {
    // do nothing;
}

let tupelo = new Tupelo(global.V8Worker2);
global.tupelo = tupelo;

export default tupelo;
