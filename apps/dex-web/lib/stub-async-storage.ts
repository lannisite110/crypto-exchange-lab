/** Stub for MetaMask SDK optional RN dependency (not used on web). */
const noop = async () => undefined;
export default {
  getItem: noop,
  setItem: noop,
  removeItem: noop,
};
