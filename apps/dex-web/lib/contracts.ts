export type AmmAddresses = {
  labToken: `0x${string}`;
  labUsd: `0x${string}`;
  factory: `0x${string}`;
  router: `0x${string}`;
  pair: `0x${string}`;
};

function addr(v: string | undefined): `0x${string}` | undefined {
  if (!v || !/^0x[a-fA-F0-9]{40}$/.test(v)) return undefined;
  return v as `0x${string}`;
}

export function getAmmAddresses(): AmmAddresses | null {
  const labToken = addr(process.env.NEXT_PUBLIC_LAB_TOKEN);
  const labUsd = addr(process.env.NEXT_PUBLIC_LAB_USD);
  const factory = addr(process.env.NEXT_PUBLIC_AMM_FACTORY);
  const router = addr(process.env.NEXT_PUBLIC_AMM_ROUTER);
  const pair = addr(process.env.NEXT_PUBLIC_AMM_PAIR);
  if (!labToken || !labUsd || !router || !pair) return null;
  return {
    labToken,
    labUsd,
    factory: factory ?? ("0x0000000000000000000000000000000000000000" as `0x${string}`),
    router,
    pair,
  };
}
