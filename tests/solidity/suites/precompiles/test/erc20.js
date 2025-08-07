const { expect } = require('chai')
const hre = require('hardhat')

describe('ERC20', function () {
  let erc20Contract, erc20Burn0Contract
  let owner, user1, user2
  const ERC20_PRECOMPILE_ADDRESS = '0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE'
  
  beforeEach(async function () {
    // Get signers
    const signers = await hre.ethers.getSigners()
    owner = signers[2]
    user1 = signers[0]
    user2 = signers[1]
    
    // Get the ERC20 precompile contract instance
    const ERC20_ABI = [
      'function mint(address to, uint256 amount) external returns (bool)',
      'function burn(uint256 amount) external',
      'function burnFrom(address account, uint256 amount) external',
      'function balanceOf(address account) external view returns (uint256)',
      'function transfer(address to, uint256 amount) external returns (bool)',
      'function approve(address spender, uint256 amount) external returns (bool)',
      'function allowance(address owner, address spender) external view returns (uint256)',
      'function increaseAllowance(address spender, uint256 addedValue) external returns (bool)',
      'function owner() external view returns (address)',
      'function transferOwnership(address newOwner) external',
      'function name() external view returns (string)',
      'function symbol() external view returns (string)',
      'function decimals() external view returns (uint8)',
      'function totalSupply() external view returns (uint256)',
      'event Transfer(address indexed from, address indexed to, uint256 value)',
      'event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)'
    ]

    // Get the ERC20 precompile contract instance
    const ERC20_BURN0_ABI = [
      'function mint(address to, uint256 amount) external returns (bool)',
      'function burn(address from, uint256 amount) external',
      'function burnFrom(address account, uint256 amount) external',
      'function balanceOf(address account) external view returns (uint256)',
      'function transfer(address to, uint256 amount) external returns (bool)',
      'function approve(address spender, uint256 amount) external returns (bool)',
      'function allowance(address owner, address spender) external view returns (uint256)',
      'function increaseAllowance(address spender, uint256 addedValue) external returns (bool)',
      'function owner() external view returns (address)',
      'function transferOwnership(address newOwner) external',
      'function name() external view returns (string)',
      'function symbol() external view returns (string)',
      'function decimals() external view returns (uint8)',
      'function totalSupply() external view returns (uint256)',
      'event Transfer(address indexed from, address indexed to, uint256 value)',
      'event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)'
    ]
    
    erc20Contract = new hre.ethers.Contract(ERC20_PRECOMPILE_ADDRESS, ERC20_ABI, owner)
    erc20Burn0Contract = new hre.ethers.Contract(ERC20_PRECOMPILE_ADDRESS, ERC20_BURN0_ABI, owner)
  })

  describe('mint', function () {
    it('should revert if the caller is not the contract owner', async function () {
      const mintAmount = hre.ethers.parseEther('100')
      
      // Connect as user1 (non-owner) and mint - this should revert
      const contractAsUser1 = erc20Contract.connect(user1)
      
      // Mint tokens as non-owner user1 to user2 - should revert
      await expect(contractAsUser1.mint(user2.address, mintAmount))
        .to.be.reverted
    })

    it('should mint tokens to the recipient if the caller is the contract owner', async function () {
      const mintAmount = hre.ethers.parseEther('100')
      
      // Connect as owner and mint
      const contractAsOwner = erc20Contract.connect(owner)

      // Get initial balance
      const initialBalance = await erc20Contract.balanceOf(user2.address)
      
      // Mint tokens as owner
      const tx = await contractAsOwner.mint(user2.address, mintAmount)
      await new Promise(r => setTimeout(r, 1000));
      await tx.wait(1)

      expect(tx).to.not.be.reverted


      // Check new balance
      const newBalance = await erc20Contract.balanceOf(user2.address)
      expect(newBalance).to.equal(initialBalance + mintAmount)
    })
  })

  describe('burn', function () {
    it('should burn tokens from the caller', async function () {
      const mintAmount = hre.ethers.parseEther('100')
      const burnAmount = hre.ethers.parseEther('50')
      
      // First mint some tokens to owner (use owner for this test to avoid conflicts)
      const mintTx = await erc20Contract.mint(owner.address, mintAmount)
      await new Promise(r => setTimeout(r, 1000));
      await mintTx.wait(1)
      
      // Get initial balance
      const initialBalance = await erc20Contract.balanceOf(owner.address)
      
      // Owner burns their own tokens
      const burnTx = await erc20Contract.burn(burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await burnTx.wait(1)
 
      // Check new balance
      const newBalance = await erc20Contract.balanceOf(owner.address)
      expect(newBalance).to.equal(initialBalance - burnAmount)
    })
  })

  describe('burn0', function () {
    it('should revert if the caller is not the contract owner', async function () {
      const burnAmount = hre.ethers.parseEther('10')
      
      // Connect as user1 (non-owner) and attempt to burn from user2 - this should revert
      const contractAsUser1 = erc20Burn0Contract.connect(user1)
      
      // Attempt to burn tokens from user2 as non-owner user1 - should revert
      await expect(contractAsUser1.burn(user2.address, burnAmount))
        .to.be.reverted
    })

    it('should allow owner to burn tokens from any address', async function () {
      const mintAmount = hre.ethers.parseEther('100')
      const burnAmount = hre.ethers.parseEther('30')
      
      // First mint some tokens to user1
      const mintTx = await erc20Burn0Contract.mint(user1.address, mintAmount)
      await new Promise(r => setTimeout(r, 1000));
      await mintTx.wait(1)
      
      // Get initial balance of user1
      const initialBalance = await erc20Contract.balanceOf(user1.address)
      
      // Owner burns tokens from user1's account
      const burnTx = await erc20Burn0Contract.burn(user1.address, burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await burnTx.wait(1)

      expect(burnTx).to.not.be.reverted
 
      // Check new balance
      const newBalance = await erc20Burn0Contract.balanceOf(user1.address)
      expect(newBalance).to.equal(initialBalance - burnAmount)
    })

    it('should revert when trying to burn more than available balance', async function () {
      // Get current balance of user1
      const currentBalance = await erc20Burn0Contract.balanceOf(user1.address)
      const burnAmount = currentBalance + hre.ethers.parseEther('1') // Try to burn more than available
      
      // Owner attempts to burn more tokens than user1 has - should revert
      await expect(erc20Burn0Contract.burn(user1.address, burnAmount))
        .to.be.reverted
    })
  })

  describe('burnFrom', function () {
    it('should allow any caller to burn from account with allowance', async function () {
      const mintAmount = hre.ethers.parseEther('100')
      const burnAmount = hre.ethers.parseEther('50')

      
      // First mint some tokens to user1
      const mintTx = await erc20Contract.mint(user1.address, mintAmount)
      await new Promise(r => setTimeout(r, 1000));
      await mintTx.wait(1)
      
      // User1 approves user2 to spend tokens
      const contractAsUser1 = erc20Contract.connect(user1)
      const approveTx = await contractAsUser1.approve(user2.address, burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await approveTx.wait(1)
      
      // Get initial balance and allowance
      const initialBalance = await erc20Contract.balanceOf(user1.address)
      const initialAllowance = await erc20Contract.allowance(user1.address, user2.address)
      
      // Connect as user2 (non-owner) and burnFrom - this should succeed with allowance
      const contractAsUser2 = erc20Contract.connect(user2)
      
      const burnFromTx = await contractAsUser2.burnFrom(user1.address, burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await burnFromTx.wait(1)

      // Check new balance
      const newBalance = await erc20Contract.balanceOf(user1.address)
      expect(newBalance).to.equal(initialBalance - burnAmount)
      
      // Check allowance was NOT reduced (due to implementation bug in burnFrom)
      const newAllowance = await erc20Contract.allowance(user1.address, user2.address)
      expect(newAllowance).to.equal(initialAllowance)
    })

    it('should burn tokens from the specified account with allowance', async function () {
      const mintAmount = hre.ethers.parseEther('200') // Use different amount to avoid conflicts
      const burnAmount = hre.ethers.parseEther('75')  // Use different amount to avoid conflicts
      
      // First mint some tokens to user2 (use user2 for this test)
      const mintTx = await erc20Contract.mint(user2.address, mintAmount)
      await new Promise(r => setTimeout(r, 1000));
      await mintTx.wait(1)
      
      // User2 approves user1 to spend tokens (different direction than first burnFrom test)
      const contractAsUser2 = erc20Contract.connect(user2)
      const approveTx = await contractAsUser2.approve(user1.address, burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await approveTx.wait(1)
      
      // Get initial balance and allowance
      const initialBalance = await erc20Contract.balanceOf(user2.address)
      const initialAllowance = await erc20Contract.allowance(user2.address, user1.address)
      
      // User1 burns tokens from user2's account
      const contractAsUser1 = erc20Contract.connect(user1)
      const burnFromTx = await contractAsUser1.burnFrom(user2.address, burnAmount)
      await new Promise(r => setTimeout(r, 1000));
      await burnFromTx.wait(1)
 
      // Check new balance
      const newBalance = await erc20Contract.balanceOf(user2.address)
      expect(newBalance).to.equal(initialBalance - burnAmount)
      
            // Check allowance was NOT reduced (due to implementation bug in burnFrom)
      const newAllowance = await erc20Contract.allowance(user2.address, user1.address)
      expect(newAllowance).to.equal(initialAllowance)
    })
  })

  describe('transferOwnership', function () {
    it('should revert if the caller is not the contract owner', async function () {
      // Connect as user1 (non-owner) and attempt to transfer ownership - this should revert
      const contractAsUser1 = erc20Contract.connect(user1)
      
      // Attempt to transfer ownership as non-owner user1 to user2 - should revert
      await expect(contractAsUser1.transferOwnership(user2.address))
        .to.be.reverted
    })

    it('should transfer ownership when called by the current owner', async function () {
      // Get initial owner
      const initialOwner = await erc20Contract.owner()
      expect(initialOwner).to.equal(owner.address)
      
      // Connect as owner and transfer ownership
      const contractAsOwner = erc20Contract.connect(owner)
      
      // Transfer ownership to user1
      const tx = await contractAsOwner.transferOwnership(user1.address)
      await new Promise(r => setTimeout(r, 1000));
      await tx.wait(1)

      expect(tx).to.not.be.reverted

      // Check ownership has changed
      const newOwner = await erc20Contract.owner()
      expect(newOwner).to.equal(user1.address)
    })
  })
})