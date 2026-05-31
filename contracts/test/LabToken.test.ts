import { expect } from "chai";
import { ethers } from "hardhat";

describe("LabToken", function () {
  it("mints initial supply to deployer", async function () {
    const [owner] = await ethers.getSigners();
    const LabToken = await ethers.getContractFactory("LabToken");
    const token = await LabToken.deploy();
    const balance = await token.balanceOf(owner.address);
    expect(balance).to.equal(ethers.parseEther("1000000"));
  });
});
