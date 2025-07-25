// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract MinimalContract {
    string public name = "MinimalContract";
    uint256 public value = 42;
    
    function getValue() public view returns (uint256) {
        return value;
    }
    
    function setValue(uint256 newValue) public {
        value = newValue;
    }
    
    function getName() public view returns (string memory) {
        return name;
    }
} 