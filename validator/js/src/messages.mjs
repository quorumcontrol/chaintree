import utils from './utils.mjs';

let messages = {}

messages.toAny = async function(typeName, msg) {
    let serialized = await utils.serialize(msg);
    let any = {
        type: typeName,
        payload: serialized,
    }
    serialized = await utils.serialize(any);
    return serialized;
}

// type Finished struct {
//	 Result []byte
// }

export default messages;