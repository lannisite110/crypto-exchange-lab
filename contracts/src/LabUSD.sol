// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @title LabUSD — mock stablecoin for AMM pairs on Sepolia (no real value)
contract LabUSD is ERC20 {
    constructor() ERC20("Lab USD", "LUSD") {
        _mint(msg.sender, 1_000_000 * 10 ** decimals());
    }
}
