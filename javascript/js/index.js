import Tupelo from './src/tupelo.mjs';

global.onerror = function(message, source, lineno, colno, error) {
    V8Worker2.print("msg: ", message, " source: ", source, " lineno: ", lineno, " err: ", error); 
}

global.setImmediate = (fn) => {
    fn();
}

global.clearImmediate = (imm) => {
    // do nothing;
}


async function hi() {
   return "hi";
}

async function tester() {
    V8Worker2.print("awaitin hi");
    var hella;
    try {
        hella = await hi();
        V8Worker2.print("hi");
    } catch(e) {
        console.error("hi caught error");
        V8Worker2.print("caught error");
    }
    return hella;
}

tester().then((resp)=> {V8Worker2.print("success")}, (err) => {V8Worker2.print("outer error: ", err)});

let tupelo = new Tupelo(global.V8Worker2);

export default tupelo;
