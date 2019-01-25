import utils from './utils';

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

export default messages;