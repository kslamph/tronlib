// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ITRC20 {
	function totalSupply() external view returns (uint256);
	function balanceOf(address account) external view returns (uint256);
	function transfer(address recipient, uint256 amount) external returns (bool);
	function allowance(address owner, address spender) external view returns (uint256);
	function approve(address spender, uint256 amount) external returns (bool);
	function transferFrom(address sender, address recipient, uint256 amount) external returns (bool);

	event Transfer(address indexed from, address indexed to, uint256 value);
	event Approval(address indexed owner, address indexed spender, uint256 value);
}

contract TRC20 is ITRC20 {
	mapping(address => uint256) private _balances;
	mapping(address => mapping(address => uint256)) private _allowances;

	uint256 private _totalSupply;
	string private _name;
	string private _symbol;
	uint8 private _decimals;

	constructor(string memory name_, string memory symbol_, uint8 decimals_, uint256 initialSupply_) {
		_name = name_;
		_symbol = symbol_;
		_decimals = decimals_;
		_mint(msg.sender, initialSupply_);
	}

	function name() public view returns (string memory) {
		return _name;
	}

	function symbol() public view returns (string memory) {
		return _symbol;
	}

	function decimals() public view returns (uint8) {
		return _decimals;
	}

	function totalSupply() public view override returns (uint256) {
		return _totalSupply;
	}

	function balanceOf(address account) public view override returns (uint256) {
		return _balances[account];
	}

	function transfer(address recipient, uint256 amount) public override returns (bool) {
		_transfer(msg.sender, recipient, amount);
		return true;
	}

	function allowance(address owner, address spender) public view override returns (uint256) {
		return _allowances[owner][spender];
	}

	function approve(address spender, uint256 amount) public override returns (bool) {
		_approve(msg.sender, spender, amount);
		return true;
	}

	function transferFrom(address sender, address recipient, uint256 amount) public override returns (bool) {
		_transfer(sender, recipient, amount);
		
		uint256 currentAllowance = _allowances[sender][msg.sender];
		require(currentAllowance >= amount, "TRC20: transfer amount exceeds allowance");
		unchecked {
			_approve(sender, msg.sender, currentAllowance - amount);
		}

		return true;
	}

	function increaseAllowance(address spender, uint256 addedValue) public returns (bool) {
		_approve(msg.sender, spender, _allowances[msg.sender][spender] + addedValue);
		return true;
	}

	function decreaseAllowance(address spender, uint256 subtractedValue) public returns (bool) {
		uint256 currentAllowance = _allowances[msg.sender][spender];
		require(currentAllowance >= subtractedValue, "TRC20: decreased allowance below zero");
		unchecked {
			_approve(msg.sender, spender, currentAllowance - subtractedValue);
		}
		return true;
	}

	function _transfer(address sender, address recipient, uint256 amount) internal {
		require(sender != address(0), "TRC20: transfer from the zero address");
		require(recipient != address(0), "TRC20: transfer to the zero address");

		uint256 senderBalance = _balances[sender];
		require(senderBalance >= amount, "TRC20: transfer amount exceeds balance");
		unchecked {
			_balances[sender] = senderBalance - amount;
		}
		_balances[recipient] += amount;

		emit Transfer(sender, recipient, amount);
	}

	function _mint(address account, uint256 amount) internal {
		require(account != address(0), "TRC20: mint to the zero address");

		_totalSupply += amount;
		_balances[account] += amount;
		emit Transfer(address(0), account, amount);
	}

	function _burn(address account, uint256 amount) internal {
		require(account != address(0), "TRC20: burn from the zero address");

		uint256 accountBalance = _balances[account];
		require(accountBalance >= amount, "TRC20: burn amount exceeds balance");
		unchecked {
			_balances[account] = accountBalance - amount;
		}
		_totalSupply -= amount;

		emit Transfer(account, address(0), amount);
	}

	function _approve(address owner, address spender, uint256 amount) internal {
		require(owner != address(0), "TRC20: approve from the zero address");
		require(spender != address(0), "TRC20: approve to the zero address");

		_allowances[owner][spender] = amount;
		emit Approval(owner, spender, amount);
	}
}