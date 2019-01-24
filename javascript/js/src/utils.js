const dagCBOR = require('ipld-dag-cbor');

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

module.exports = utils;