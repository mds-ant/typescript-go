//// [tests/cases/compiler/jsdocTagInDiscardedArrowSpeculation.ts] ////

//// [typedef.js]
const a = 1, b = 2;
export const r = (/** @typedef {object} L4 */ a) - b;
/** @type {L4} */
export const q = {};

//// [callback.js]
const c = 1, d = 2;
export const s = (/** @callback CB4
 * @param {number} x
 * @returns {number}
 */ c) - d;
/** @type {CB4} */
export const t = (x) => x;

//// [types.ts]
export type T4 = { n: number };

//// [import.js]
const e = 1, f = 2;
export const u = (/** @import {T4} from "./types" */ e) - f;
/** @type {T4} */
export const v = { n: 1 };




//// [typedef.d.ts]
export declare const r: number;
/** @type {L4} */
export declare const q: L4;
//// [callback.d.ts]
export declare const s: number;
/** @type {CB4} */
export declare const t: CB4;
//// [types.d.ts]
export type T4 = {
    n: number;
};
//// [import.d.ts]
export declare const u: number;
/** @type {T4} */
export declare const v: T4;
