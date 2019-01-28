class SimpleValidator {

    async started(tupelo) {
        this.tupelo = tupelo;
        let resp = await tupelo.nodestore.resolve(tupelo.tip, "tree/ok")
        if (resp.value) {
            this.tupelo.send("finished", {
                result: "ok",
            });
            return
        } 
        this.tupelo.send("finished", {
            result: "invalid",
        });
    }
}

let sv = new SimpleValidator();

tupelo.onStart(sv.started);