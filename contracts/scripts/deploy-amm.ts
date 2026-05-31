import * as fs from "fs";
import * as path from "path";
import { ethers, network } from "hardhat";

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deployer:", deployer.address);
  console.log("Network:", network.name);

  const LabToken = await ethers.getContractFactory("LabToken");
  const lab = await LabToken.deploy();
  await lab.waitForDeployment();

  const LabUSD = await ethers.getContractFactory("LabUSD");
  const lusd = await LabUSD.deploy();
  await lusd.waitForDeployment();

  const Factory = await ethers.getContractFactory("UniswapV2Factory");
  const factory = await Factory.deploy(deployer.address);
  await factory.waitForDeployment();

  const factoryAddr = await factory.getAddress();
  const Router = await ethers.getContractFactory("UniswapV2Router");
  const router = await Router.deploy(factoryAddr);
  await router.waitForDeployment();

  const labAddr = await lab.getAddress();
  const lusdAddr = await lusd.getAddress();
  const routerAddr = await router.getAddress();

  await factory.createPair(labAddr, lusdAddr);
  const pairAddr = await factory.getPair(labAddr, lusdAddr);

  const net = await ethers.provider.getNetwork();
  const deployment = {
    network: network.name,
    chainId: net.chainId.toString(),
    deployer: deployer.address,
    labToken: labAddr,
    labUsd: lusdAddr,
    factory: factoryAddr,
    router: routerAddr,
    pair: pairAddr,
    deployedAt: new Date().toISOString(),
  };

  if (network.name === "hardhat" || network.name === "localhost") {
    const labSeed = ethers.parseEther("10000");
    const lusdSeed = ethers.parseEther("10000");
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
    console.log("Seeded LAB/LUSD pool on local network");
  }

  const outDir = path.join(__dirname, "..", "deployments");
  fs.mkdirSync(outDir, { recursive: true });
  const outFile = path.join(outDir, `${network.name}.json`);
  fs.writeFileSync(outFile, JSON.stringify(deployment, null, 2));
  console.log("Wrote", outFile);
  console.log(JSON.stringify(deployment, null, 2));
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
