import { JSONEncoder, JSON } from "assemblyscript-json";

// Create encoder
let encoder = new JSONEncoder();

// Construct necessary object
encoder.pushObject("obj");
encoder.setInteger("int", 10);
encoder.setString("str", "");
encoder.popObject();

// Get serialized data
let json: Uint8Array = encoder.serialize();

// Or get serialized data as string
let jsonString: string = encoder.toString();

assert(jsonString, '"obj": {"int": 10, "str": ""}'); // True!
console.log(jsonString)

let d = (<JSON.Obj>JSON.parse('{"int": 10, "str": "1"}'));
console.log(d.getInteger('int')!.stringify());
console.log(d.getString('str')!.stringify());
