// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Enum definition
enum Status {
    Pending,
    Approved,
    Rejected
}

/// @title TestComprehensiveTypes - A contract for testing encoder/decoder with diverse types and events, without struct/tuple types.
contract TestComprehensiveTypes {
    // State variables for primitive types
    uint8 public myUint8;
    int8 public myInt8;
    uint256 public myUint256;
    int256 public myInt256;
    address public myAddress;
    bool public myBool;
    string public myString;
    bytes public myBytes;
    bytes32 public myBytes32;

    // State variables for array types
    uint256[] public uintArray;
    address[] public addressArray;
    string[] public stringArray;
    bytes[] public bytesArray;
    bool[3] public fixedBoolArray; // Fixed-size array

    // State variables for enum (no struct)
    Status public currentStatus;

    // Events to log various types
    event PrimitiveTypesEvent(
        uint8 indexed u8,
        int8 i8,
        uint256 u256,
        int256 i256,
        address indexed addr,
        bool b,
        string s,
        bytes bs,
        bytes32 b32
    );
    event ArrayTypesEvent(
        uint256[] uArray,
        address[] addrArray,
        string[] sArray,
        bytes[] bArray,
        bool[3] fixedBArray
    );
    event EnumEvent(Status indexed status);
    event MixedArrayEvent(uint256[] indexed numbers, address[] addresses);


    // --- Constructor ---
    constructor(
        uint8 _myUint8,
        int8 _myInt8,
        uint256 _myUint256,
        int256 _myInt256,
        address _myAddress,
        bool _myBool,
        string memory _myString,
        bytes memory _myBytes,
        bytes32 _myBytes32,
        uint256[] memory _uintArray,
        address[] memory _addressArray,
        string[] memory _stringArray,
        bytes[] memory _bytesArray,
        bool[3] memory _fixedBoolArray,
        Status _currentStatus
    ) {
        myUint8 = _myUint8;
        myInt8 = _myInt8;
        myUint256 = _myUint256;
        myInt256 = _myInt256;
        myAddress = _myAddress;
        myBool = _myBool;
        myString = _myString;
        myBytes = _myBytes;
        myBytes32 = _myBytes32;
        uintArray = _uintArray;
        addressArray = _addressArray;
        stringArray = _stringArray;
        bytesArray = _bytesArray;
        fixedBoolArray = _fixedBoolArray;
        currentStatus = _currentStatus;
    }

    // --- Functions for setting primitive types ---
    function setPrimitiveTypes(
        uint8 _u8,
        int8 _i8,
        uint256 _u256,
        int256 _i256,
        address _addr,
        bool _b,
        string memory _s,
        bytes memory _bs,
        bytes32 _b32
    ) public {
        myUint8 = _u8;
        myInt8 = _i8;
        myUint256 = _u256;
        myInt256 = _i256;
        myAddress = _addr;
        myBool = _b;
        myString = _s;
        myBytes = _bs;
        myBytes32 = _b32;
        emit PrimitiveTypesEvent(_u8, _i8, _u256, _i256, _addr, _b, _s, _bs, _b32);
    }

    // --- Functions for getting primitive types ---
    function getUint8() public view returns (uint8) { return myUint8; }
    function getInt8() public view returns (int8) { return myInt8; }
    function getUint256() public view returns (uint256) { return myUint256; }
    function getInt256() public view returns (int256) { return myInt256; }
    function getAddress() public view returns (address) { return myAddress; }
    function getBool() public view returns (bool) { return myBool; }
    function getString() public view returns (string memory) { return myString; }
    function getBytes() public view returns (bytes memory) { return myBytes; }
    function getBytes32() public view returns (bytes32) { return myBytes32; }

    // --- Functions for setting array types ---
    function setArrayTypes(
        uint256[] memory _uArray,
        address[] memory _addrArray,
        string[] memory _sArray,
        bytes[] memory _bArray,
        bool[3] memory _fixedBArray
    ) public {
        uintArray = _uArray;
        addressArray = _addrArray;
        stringArray = _sArray;
        bytesArray = _bArray;
        fixedBoolArray = _fixedBArray;
        emit ArrayTypesEvent(_uArray, _addrArray, _sArray, _bArray, _fixedBArray);
    }

    // --- Functions for getting array types ---
    function getUintArray() public view returns (uint256[] memory) { return uintArray; }
    function getAddressArray() public view returns (address[] memory) { return addressArray; }
    function getStringArray() public view returns (string[] memory) { return stringArray; }
    function getBytesArray() public view returns (bytes[] memory) { return bytesArray; }
    function getFixedBoolArray() public view returns (bool[3] memory) { return fixedBoolArray; }

    // --- Functions for setting enum ---
    function setStatus(Status _status) public {
        currentStatus = _status;
        emit EnumEvent(_status);
    }

    // --- Functions for getting enum ---
    function getStatus() public view returns (Status) { return currentStatus; }

    // --- Functions with multiple return values ---
    function getMixedPrimitives() public pure returns (
        uint8,
        address,
        bool,
        string memory,
        bytes32
    ) {
        return (
            10,
            0x742d35Cc6634C0532925a3b844Bc454e4438f44e,
            true,
            "Hello Mixed",
            0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
        );
    }

    function getMixedArrays() public pure returns (
        uint256[] memory,
        string[] memory,
        bytes[] memory
    ) {
        uint256[] memory _uArray = new uint256[](2);
        _uArray[0] = 100;
        _uArray[1] = 200;

        string[] memory _sArray = new string[](2);
        _sArray[0] = "First";
        _sArray[1] = "Second";

        bytes[] memory _bArray = new bytes[](1);
        _bArray[0] = hex"aabbcc";

        return (_uArray, _sArray, _bArray);
    }

    function getMixedStructAndEnum() public pure returns (Status) {
        return Status.Approved;
    }

    function emitMixedArrayEvent(uint256[] memory numbers, address[] memory addresses) public {
        emit MixedArrayEvent(numbers, addresses);
    }
}