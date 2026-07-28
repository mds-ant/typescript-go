// @allowJs: true
// @checkJs: true
// @declaration: true
// @emitDeclarationOnly: true

// @filename: typedef.js
const a = 1, b = 2;
export const r = (/** @typedef {object} L4 */ a) - b;
/** @type {L4} */
export const q = {};

// @filename: callback.js
const c = 1, d = 2;
export const s = (/** @callback CB4
 * @param {number} x
 * @returns {number}
 */ c) - d;
/** @type {CB4} */
export const t = (x) => x;

// @filename: types.ts
export type T4 = { n: number };

// @filename: import.js
const e = 1, f = 2;
export const u = (/** @import {T4} from "./types" */ e) - f;
/** @type {T4} */
export const v = { n: 1 };
