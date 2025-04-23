// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract TestContract {
    uint256 public counter;

    function incrementCounter() public {
        counter++;
    }

    function getCounter() public view returns (uint256) {
        return counter;
    }

    function greet(string memory name) public pure returns (string memory) {
        return string(abi.encodePacked("Hello, ", name, "!"));
    }

    function isEven(uint256 number) public pure returns (bool) {
        return number % 2 == 0;
    }

    function getThisAddress() public view returns (address) {
        return address(this);
    }

    function getUserInfo() public pure returns (uint256, string memory, bool) {
        return (123, "Test User", true);
    }

    function getComplexInfo() public pure returns (uint256, string memory, address) {
        address complexAddress = 0xa614f803B6FD780986A42c78Ec9c7f77e6DeD13C;
        return (456, "Complex Data", complexAddress);
    }

    function revertWithMessage() public pure {
        revert("This function always reverts");
    }

    function revertOnZero(uint256 value) public pure {
        require(value != 0, "Value cannot be zero");
    }

    function safeDivide(uint256 numerator, uint256 denominator) public pure returns (uint256) {
        require(denominator > 0, "Division by zero");
        return numerator / denominator;
    }

    function causeOverflow() public {
        unchecked {
            uint256 max = type(uint256).max;
            counter = max + 1;
        }
    }

    // Simple return value functions
    function getSimpleBool() public pure returns (bool) {
        return true;
    }

    function getSimpleUint8() public pure returns (uint8) {
        return 255;
    }

    function getSimpleUint256() public pure returns (uint256) {
        return 115792089237316195423570985008687907853269984665640564039457584007913129639935;
    }

    function getSimpleInt8() public pure returns (int8) {
        return -128;
    }

    function getSimpleInt256() public pure returns (int256) {
        return -57896044618658097711785492504343953926634992332820282019728792003956564819968;
    }

    function getSimpleAddress() public pure returns (address) {
        return 0x742d35Cc6634C0532925a3b844Bc454e4438f44e;
    }

    function getSimpleBytes1() public pure returns (bytes1) {
        return 0xff;
    }

    function getSimpleBytes32() public pure returns (bytes32) {
        return 0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef;
    }

    function getSimpleString() public pure returns (string memory) {
        return "Simple string return";
    }

    function getUnicodeString() public pure returns (string memory) {
        return unicode"Hello 🌍 World! ⭐";
    }

    function getHexLiteral() public pure returns (bytes memory) {
        return hex"deadbeef";
    }

    // Multiple return values with mixed types
    function getMultipleValues1() public pure returns (uint8, bytes1, string memory) {
        return (42, 0xa1, "Mixed types");
    }

    function getMultipleValues2() public pure returns (address, bool, int256, bytes32) {
        return (
            0x742d35Cc6634C0532925a3b844Bc454e4438f44e,
            true,
            -1234567890,
            0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
        );
    }

    // Nonpayable function examples
    function setMultipleValues(uint256 _counter) public {
        counter = _counter;
    }

    function complexOperation(uint256 input) public {
        if (input > 100) {
            counter = input * 2;
        } else {
            counter = input / 2;
        }
    }
}