"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  formatEther,
  parseEther,
  maxUint256,
  type Address,
} from "viem";
import {
  useAccount,
  useConnect,
  useDisconnect,
  usePublicClient,
  useReadContract,
  useWriteContract,
  useWaitForTransactionReceipt,
} from "wagmi";
import { sepolia } from "wagmi/chains";
import { erc20Abi, pairAbi, routerAbi } from "../lib/abis";
import { getAmmAddresses } from "../lib/contracts";

type AmmMode = "swap" | "liquidity";

export function AmmPanel() {
  const addresses = useMemo(() => getAmmAddresses(), []);
  const { address, isConnected, chainId } = useAccount();
  const { connect, connectors, isPending: connecting } = useConnect();
  const { disconnect } = useDisconnect();

  const [mode, setMode] = useState<AmmMode>("swap");
  const [swapDir, setSwapDir] = useState<"LAB_LUSD" | "LUSD_LAB">("LAB_LUSD");
  const [amountIn, setAmountIn] = useState("1");
  const [labAmt, setLabAmt] = useState("10");
  const [lusdAmt, setLusdAmt] = useState("10");
  const [lpRemove, setLpRemove] = useState("");
  const [error, setError] = useState("");
  const [txHash, setTxHash] = useState<`0x${string}` | undefined>();

  const publicClient = usePublicClient();
  const { writeContractAsync, isPending: writing } = useWriteContract();
  const { isLoading: confirming } = useWaitForTransactionReceipt({ hash: txHash });

  const onSepolia = chainId === sepolia.id;
  const router = addresses?.router;
  const lab = addresses?.labToken;
  const lusd = addresses?.labUsd;
  const pair = addresses?.pair;

  const path: Address[] | undefined =
    lab && lusd
      ? swapDir === "LAB_LUSD"
        ? [lab, lusd]
        : [lusd, lab]
      : undefined;

  const amountInWei = useMemo(() => {
    try {
      return parseEther(amountIn || "0");
    } catch {
      return BigInt(0);
    }
  }, [amountIn]);

  const { data: amountsOut } = useReadContract({
    address: router,
    abi: routerAbi,
    functionName: "getAmountsOut",
    args:
      router && path && amountInWei > BigInt(0)
        ? [amountInWei, path]
        : undefined,
    query: { enabled: !!router && !!path && amountInWei > BigInt(0) },
  });

  const { data: labBal, refetch: refetchLab } = useReadContract({
    address: lab,
    abi: erc20Abi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!lab && !!address },
  });

  const { data: lusdBal, refetch: refetchLusd } = useReadContract({
    address: lusd,
    abi: erc20Abi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!lusd && !!address },
  });

  const { data: lpBal, refetch: refetchLp } = useReadContract({
    address: pair,
    abi: pairAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!pair && !!address },
  });

  const { data: reserves } = useReadContract({
    address: pair,
    abi: pairAbi,
    functionName: "getReserves",
    query: { enabled: !!pair },
  });

  const { data: token0 } = useReadContract({
    address: pair,
    abi: pairAbi,
    functionName: "token0",
    query: { enabled: !!pair },
  });

  const refreshBalances = useCallback(() => {
    refetchLab();
    refetchLusd();
    refetchLp();
  }, [refetchLab, refetchLusd, refetchLp]);

  useEffect(() => {
    if (!confirming && txHash) {
      refreshBalances();
      setTxHash(undefined);
    }
  }, [confirming, txHash, refreshBalances]);

  const reserveLab =
    reserves && token0 && lab
      ? token0.toLowerCase() === lab.toLowerCase()
        ? reserves[0]
        : reserves[1]
      : undefined;
  const reserveLusd =
    reserves && token0 && lusd
      ? token0.toLowerCase() === lusd.toLowerCase()
        ? reserves[0]
        : reserves[1]
      : undefined;

  const deadline = BigInt(Math.floor(Date.now() / 1000) + 1200);

  async function approveMax(token: Address, spender: Address) {
    if (!publicClient) throw new Error("RPC client unavailable");
    const hash = await writeContractAsync({
      address: token,
      abi: erc20Abi,
      functionName: "approve",
      args: [spender, maxUint256],
    });
    setTxHash(hash);
    await publicClient.waitForTransactionReceipt({ hash });
  }

  async function onSwap(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!addresses || !router || !path || !address) {
      setError("Set NEXT_PUBLIC_* AMM addresses (deploy to Sepolia first).");
      return;
    }
    if (!onSepolia) {
      setError("Switch MetaMask to Sepolia.");
      return;
    }
    try {
      const out = amountsOut?.[1];
      if (!out) throw new Error("Could not quote output");
      const minOut = (out * BigInt(99)) / BigInt(100);
      await approveMax(path[0], router);
      const hash = await writeContractAsync({
        address: router,
        abi: routerAbi,
        functionName: "swapExactTokensForTokens",
        args: [amountInWei, minOut, path, address, deadline],
      });
      setTxHash(hash);
      refreshBalances();
    } catch (err) {
      setError(err instanceof Error ? err.message : "swap failed");
    }
  }

  async function onAddLiquidity(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!addresses || !router || !lab || !lusd || !address) {
      setError("Configure contract addresses in env.");
      return;
    }
    if (!onSepolia) {
      setError("Switch MetaMask to Sepolia.");
      return;
    }
    try {
      const a = parseEther(labAmt);
      const b = parseEther(lusdAmt);
      await approveMax(lab, router);
      await approveMax(lusd, router);
      const hash = await writeContractAsync({
        address: router,
        abi: routerAbi,
        functionName: "addLiquidity",
        args: [lab, lusd, a, b, BigInt(0), BigInt(0), address, deadline],
      });
      setTxHash(hash);
      refreshBalances();
    } catch (err) {
      setError(err instanceof Error ? err.message : "add liquidity failed");
    }
  }

  async function onRemoveLiquidity(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (!addresses || !router || !lab || !lusd || !pair || !address) {
      setError("Configure contract addresses in env.");
      return;
    }
    if (!onSepolia) {
      setError("Switch MetaMask to Sepolia.");
      return;
    }
    try {
      const liq = parseEther(lpRemove || "0");
      if (!publicClient) throw new Error("RPC client unavailable");
      const hashApprove = await writeContractAsync({
        address: pair,
        abi: pairAbi,
        functionName: "approve",
        args: [router, liq],
      });
      setTxHash(hashApprove);
      await publicClient.waitForTransactionReceipt({ hash: hashApprove });
      const hash = await writeContractAsync({
        address: router,
        abi: routerAbi,
        functionName: "removeLiquidity",
        args: [lab, lusd, liq, BigInt(0), BigInt(0), address, deadline],
      });
      setTxHash(hash);
      refreshBalances();
    } catch (err) {
      setError(err instanceof Error ? err.message : "remove liquidity failed");
    }
  }

  if (!addresses) {
    return (
      <section className="panel">
        <h2>AMM not configured</h2>
        <p className="muted">
          Deploy contracts with{" "}
          <code>pnpm contracts:deploy:sepolia</code>, then set{" "}
          <code>NEXT_PUBLIC_LAB_TOKEN</code>, <code>NEXT_PUBLIC_LAB_USD</code>,{" "}
          <code>NEXT_PUBLIC_AMM_ROUTER</code>, and <code>NEXT_PUBLIC_AMM_PAIR</code>{" "}
          in your env. See <code>docs/amm-dex.md</code>.
        </p>
      </section>
    );
  }

  return (
    <>
      <p className="muted">
        Uniswap V2–style constant-product pool on Sepolia (LAB / LUSD). Test
        tokens only — no real value.
      </p>

      <section className="panel row wallet-bar">
        {isConnected ? (
          <>
            <span className="muted small">
              {address?.slice(0, 6)}…{address?.slice(-4)}
              {!onSepolia && " · wrong network"}
            </span>
            <button type="button" className="link" onClick={() => disconnect()}>
              Disconnect
            </button>
          </>
        ) : (
          <button
            type="button"
            onClick={() => connect({ connector: connectors[0] })}
            disabled={connecting}
          >
            {connecting ? "Connecting…" : "Connect wallet"}
          </button>
        )}
      </section>

      {error && <p className="error">{error}</p>}
      {(writing || confirming) && (
        <p className="muted small">Transaction pending…</p>
      )}

      <div className="tabs sub">
        <button
          type="button"
          className={mode === "swap" ? "active" : ""}
          onClick={() => setMode("swap")}
        >
          Swap
        </button>
        <button
          type="button"
          className={mode === "liquidity" ? "active" : ""}
          onClick={() => setMode("liquidity")}
        >
          Liquidity
        </button>
      </div>

      <div className="grid">
        <section className="panel">
          <h2>Wallet balances</h2>
          <ul className="balances-list">
            <li>LAB: {labBal != null ? formatEther(labBal) : "—"}</li>
            <li>LUSD: {lusdBal != null ? formatEther(lusdBal) : "—"}</li>
            <li>LP (CEL-LP): {lpBal != null ? formatEther(lpBal) : "—"}</li>
          </ul>
          <h3 className="small">Pool reserves</h3>
          <ul className="balances-list">
            <li>LAB: {reserveLab != null ? formatEther(reserveLab) : "—"}</li>
            <li>LUSD: {reserveLusd != null ? formatEther(reserveLusd) : "—"}</li>
          </ul>
        </section>

        {mode === "swap" ? (
          <section className="panel">
            <h2>Swap</h2>
            <form onSubmit={onSwap} className="form">
              <label>
                Direction
                <select
                  value={swapDir}
                  onChange={(e) =>
                    setSwapDir(e.target.value as "LAB_LUSD" | "LUSD_LAB")
                  }
                >
                  <option value="LAB_LUSD">LAB → LUSD</option>
                  <option value="LUSD_LAB">LUSD → LAB</option>
                </select>
              </label>
              <label>
                Amount in
                <input
                  value={amountIn}
                  onChange={(e) => setAmountIn(e.target.value)}
                />
              </label>
              <p className="muted small">
                Est. out:{" "}
                {amountsOut?.[1] != null
                  ? formatEther(amountsOut[1])
                  : "—"}{" "}
                (0.3% fee)
              </p>
              <button type="submit" disabled={!isConnected || writing}>
                Swap
              </button>
            </form>
          </section>
        ) : (
          <section className="panel">
            <h2>Add liquidity</h2>
            <form onSubmit={onAddLiquidity} className="form">
              <label>
                LAB
                <input value={labAmt} onChange={(e) => setLabAmt(e.target.value)} />
              </label>
              <label>
                LUSD
                <input value={lusdAmt} onChange={(e) => setLusdAmt(e.target.value)} />
              </label>
              <button type="submit" disabled={!isConnected || writing}>
                Add liquidity
              </button>
            </form>
            <h2>Remove liquidity</h2>
            <form onSubmit={onRemoveLiquidity} className="form">
              <label>
                LP amount
                <input
                  value={lpRemove}
                  onChange={(e) => setLpRemove(e.target.value)}
                  placeholder={lpBal != null ? formatEther(lpBal) : "0"}
                />
              </label>
              <button type="submit" disabled={!isConnected || writing}>
                Remove liquidity
              </button>
            </form>
          </section>
        )}
      </div>

      <p className="muted small">
        Router: {router?.slice(0, 10)}… · Pair: {pair?.slice(0, 10)}…
      </p>
    </>
  );
}
