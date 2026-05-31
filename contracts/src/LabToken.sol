// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// @title LabToken — test-only ERC20 for Crypto Exchange Lab (no real value)
contract LabToken is ERC20 {
    constructor() ERC20("Lab Token", "LAB") {
        _mint(msg.sender, 1_000_000 * 10 ** decimals());
    }
}
