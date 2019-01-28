class SimpleValidator {

    started(tupelo) {
        this.tupelo = tupelo;
        this.tupelo.send("finished", {
            result: "ok",
        });
    }
}

let sv = new SimpleValidator();

tupelo.onStart(sv.started);