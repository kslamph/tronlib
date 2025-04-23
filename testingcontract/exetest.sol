// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract ReturnVariables {
    // State variables to store the values
    address public myAddress;
    bool public myBool;
    uint256 public myUint;

    // Constructor to initialize the state variables
    constructor(address _myAddress, bool _myBool, uint256 _myUint) {
        myAddress = _myAddress;
        myBool = _myBool;
        myUint = _myUint;
    }

    // Function to return the three variables and update state
    function getAndUpdateVariables(address _newAddress, bool _newBool, uint256 _newUint) public returns (address, bool, uint256) {
        // Update the state variables with the new values
        myAddress = _newAddress;
        myBool = _newBool;
        myUint = _newUint;

        // Return the updated values
        return (myAddress, myBool, myUint);
    }
}