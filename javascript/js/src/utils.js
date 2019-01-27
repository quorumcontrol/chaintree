import dagCBOR from 'ipld-dag-cbor';
import multihashing from 'multihashing-async';
import CID from 'cids';

const utils = {};

utils.serialize = function(obj) {
    return new Promise((resolve, reject) => {
        dagCBOR.util.serialize(obj, (err, serialized) => {
            if (err) {
                reject(err);
                return
            }
            resolve(serialized);
        });
    });   
};

utils.deserialize = function(serialized) {
    return new Promise((resolve, reject) => {
        dagCBOR.util.deserialize(serialized, (err, obj) => {
            if (err) {
                reject(err);
                return
            }
            resolve(obj);
        });
    });   
}

utils.cid = async function(obj) {
    let serialized = await utils.serialize(obj);
    return await utils.cidOfSerialized(serialized);
}

utils.cidOfSerialized = function(serialized) {
        const hashAlg = dagCBOR.resolver.defaultHashAlg
        const hashLen = null
        const version = 1
        return new Promise((resolve, reject) => {
            multihashing(serialized, hashAlg, hashLen, (err, mh) => {
                if (err) {
                    reject(err);
                    return
                }
                resolve(new CID(version, dagCBOR.resolver.multicodec, mh));
              });
        });
}

utils.resolve = async function(blob, path) {
    return new Promise((resolve,reject)=> {
        dagCBOR.resolver.resolve(blob, path, (err,resp) => {
            if (err) {
                reject(err);
                return
            }

            resolve(resp);
        });
    });
}

export default utils;