import { expect } from "chai";
import { ethers } from "hardhat";

describe("AMM (Uniswap V2 style)", function () {
  it("mints liquidity, swaps LAB→LUSD, removes liquidity", async function () {
    const [deployer, trader] = await ethers.getSigners();

    const LabToken = await ethers.getContractFactory("LabToken");
    const lab = await LabToken.deploy();
    const LabUSD = await ethers.getContractFactory("LabUSD");
    const lusd = await LabUSD.deploy();

    const Factory = await ethers.getContractFactory("UniswapV2Factory");
    const factory = await Factory.deploy(deployer.address);
    const Router = await ethers.getContractFactory("UniswapV2Router");
    const router = await Router.deploy(await factory.getAddress());

    const labAddr = await lab.getAddress();
    const lusdAddr = await lusd.getAddress();
    const routerAddr = await router.getAddress();

    const labSeed = ethers.parseEther("10000");
    const lusdSeed = ethers.parseEther("20000");
    const deadline = BigInt(Math.floor(Date.now() / 1000) + 3600);

    await lab.approve(routerAddr, labSeed);
    await lusd.approve(routerAddr, lusdSeed);
    await router.addLiquidity(
      labAddr,
      lusdAddr,
      labSeed,
      lusdSeed,
      0,
      0,
      deployer.address,
      deadline
    );

    const pairAddr = await factory.getPair(labAddr, lusdAddr);
    expect(pairAddr).to.properAddress;

    const Pair = await ethers.getContractFactory("UniswapV2Pair");
    const pair = Pair.attach(pairAddr);
    const [r0, r1] = await pair.getReserves();
    expect(r0).to.be.gt(0);
    expect(r1).to.be.gt(0);

    const swapIn = ethers.parseEther("100");
    await lab.transfer(trader.address, swapIn);
    await lab.connect(trader).approve(routerAddr, swapIn);

    const amountsOut = await router.getAmountsOut(swapIn, [labAddr, lusdAddr]);
    const minOut = (amountsOut[1] * 99n) / 100n;

    const lusdBefore = await lusd.balanceOf(trader.address);
    await router
      .connect(trader)
      .swapExactTokensForTokens(swapIn, minOut, [labAddr, lusdAddr], trader.address, deadline);
    const lusdAfter = await lusd.balanceOf(trader.address);
    expect(lusdAfter - lusdBefore).to.be.gte(minOut);

    const lpBal = await pair.balanceOf(deployer.address);
    await pair.approve(routerAddr, lpBal);
    await router.removeLiquidity(labAddr, lusdAddr, lpBal, 0, 0, deployer.address, deadline);
    const lpAfter = await pair.balanceOf(deployer.address);
    expect(lpAfter).to.equal(0n);
  });
});
