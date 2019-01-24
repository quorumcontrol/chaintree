const buffer = require('buffer');
const utils = require('./utils');

let messages = {}

messages.toAny = async function(typeName, msg) {
    let serialized = await utils.serialize(msg);
    any = {
        type: typeName,
        payload: serialized,
    }
    serialized = await utils.serialize(any);
    return serialized;
}

module.exports = messages;