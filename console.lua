-- Simple bank account manager for transactions in games

--[[ 

Accounts are the base of the system.
They are identified by a unique integer and tied to a holder.
The account does not have a balance, but tracks a list of transactions.
This transactions can be credits, or debits.

--]]

-- ERROR CODES
NEMO = 10000 -- account does not exist
TIMETRAVEL = 10001 -- due date of transaction already happened befor creation
FLIPIT = 10002 -- the amount should never be negative, make the creditor the debitor
-- The abstract date functions (think of it as atomic slices of time)


-- Accounts and transactions are stored in a local sqlite database
local sqlite3 = require("lsqlite3complete")

-- System table 
local system_table = [[
CREATE TABLE IF NOT EXISTS system ( 
id INTEGER NOT NULL PRIMARY KEY, 
name VARCHAR(100) NOT NULL,
clock INTEGER);
]]

-- Accounts table
local accounts_table = [[
CREATE TABLE IF NOT EXISTS accounts ( 
id INTEGER NOT NULL PRIMARY KEY, 
holder VARCHAR(100) NOT NULL,
date INTEGER NOT NULL);
]]

-- Transactions table
local transactions_table = [[
CREATE TABLE IF NOT EXISTS transactions ( 
id INTEGER NOT NULL PRIMARY KEY, 
creditor INTEGER NOT NULL,
debitor INTEGER NOT NULL,
amount INTEGER NOT NULL,
concept TEXT,
date_created INTEGER NOT NULL,
date_due INTEGER NOT NULL,
payed BOOLEAN NOT NULL DEFAULT FALSE,
revoked BOOLEAN NOT NULL DEFAULT FALSE,
FOREIGN KEY (creditor) REFERENCES accounts(id),
FOREIGN KEY (debitor) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS letters ( 
    id INTEGER PRIMARY KEY, 
    sender INTEGER NOT NULL, 
    receiver INTEGER NOT NULL,
    Title VARCHAR(255),
    Path VARCHAR(2048),
    Date INTEGER,
    FOREIGN KEY (sender) REFERENCES accounts (id),
    FOREIGN KEY (receiver) REFERENCES accounts (id)
);
]]

local function create_bank(bank_name)
    local bank = sqlite3.open(bank_name)
    local err = bank:exec(system_table .. accounts_table .. transactions_table)
    if err ~= sqlite3.OK then print("Error creating bank: " .. bank:errmsg()) end
    local err = bank:exec("INSERT INTO system (name, clock) VALUES(" .. "\"" .. bank_name .. "\"," .. 0 .. ");")
    bank:close()
end

function returning(udata, cols, values, names)
    for i=1,cols do print(names[i], values[i]) end
    return 0
end

-- Entry

Bank = {}

function Bank:new(bank_name)
    local bank = {
        name = bank_name,
        db_filename = bank_name .. ".sqlite3",
        db = nil
    }

    create_bank(bank.db_filename)

    self.__index = self

    return setmetatable(bank, self)
end

function Bank:insert(table_name, fields, values) 
    local db = sqlite3.open(self.db_filename)
    local err = db:exec("INSERT INTO " .. table_name .. " (" .. fields .. ") VALUES(" .. values .. ");")
    if err ~= sqlite3.OK then print("Error on insert: " .. db:errmsg()) end
    db:close()
    return err
end

function Bank:update(table_name, field, value1, where, value2) 
    local db = sqlite3.open(self.db_filename)
    local err = db:exec("UPDATE " .. table_name .. " SET " .. field .. " = " .. value1 .. " WHERE " .. where .. " = " .. value2 .. ";")
    if err ~= sqlite3.OK then print("Error on update: " .. db:errmsg()) end
    db:close()
    return err
end


function Bank:select(table_name, field, value1, where, value2) 
    local db = sqlite3.open(self.db_filename)
    local err = db:nrows("SELECT " .. field " FROM " .. table_name .. " SET " .. field .. " = " .. value1 .. " WHERE " .. where .. " = " .. value2 .. ";")
    db:close()
    return err
end

function Bank:thisDate()
    local db = sqlite3.open(self.db_filename)
    for d in db:nrows("SELECT clock FROM system WHERE id = 1;") do db:close() return d.clock end --hacky...
end

function Bank:nextDate()
    local db = sqlite3.open(self.db_filename)
    local date = self:thisDate()
    db:exec("UPDATE system SET clock =" .. date + 1 .. " WHERE id = 1;"
            .."UPDATE transactions SET payed = 1 WHERE date_due >= (SELECT clock FROM system WHERE id = 1);")
    return date + 1
end


function Bank:createTransaction(creditor_id, debitor_id, amount, concept, due, payed) 
    if creditor_id == nil or debitor_id == nil then return NEMO end
    local date = self:thisDate()
    if due < date then return TIMETRAVEL end
    if amount < 0 then return FLIPIT end

    local err = self:insert(
        "transactions",
        "creditor, debitor, amount, concept, date_created, date_due, payed, revoked",
        creditor_id .. ", " .. 
        debitor_id .. ", " .. 
        amount .. ", " .. 
        "\"" .. concept .. "\"" .. ", " ..
        date .. ", " ..
        due .. ", " ..
        tostring(payed) .. ", false"
    )

    return err
end

function Bank:createAccount(holder_name) 
    if holder_name == nil then return NEMO end
    local date = self:thisDate()
    local sql =
    "INSERT INTO accounts (id, holder, date)"..
    "VALUES(" .. math.random(1000,9999) .. ",\"" .. holder_name .. "\"," .. date .. ");"
    
    local bank = sqlite3.open(self.db_filename)
    local err = bank:exec(sql)
    if err == sqlite3.CONSTRAINT then 
        while err == sqlite3.CONSTRAINT do
            local sql =
            "INSERT INTO accounts (id, holder, date)"..
            "VALUES(" .. math.random(1000,9999) .. ".\"" .. holder_name .. "\"," .. date .. ");"
            local err = bank:exec(sql)
        end
    elseif err ~= sqlite3.OK then
        print("Error creating account: " .. err .. bank:errmsg()) 
    end	

    local id = bank:last_insert_rowid()
    bank:close()
    
    
    local account = {
        id = id,
        holder = holder_name,
        date = date,
        bank = self
    }

    return err, account
end

function Bank:executeTransaction(transaction_id)
    return self:update("transactions", "payed", 1, "id", transaction_id)
end

function Bank:revokeTransaction(transaction_id)
    return self:update("transactions", "revoked", 1, "id", transaction_id)
end


function Bank:balance(account_id)
    local balance = {
        credits_total = 0,
        credits_payed = 0,
        credits_unpayed = 0,
        
        debits_total = 0,
        debits_payed = 0,
        debits_unpayed = 0,

        balance_total = 0,
        cash = 0,
        debt_incurred = 0, -- debt not possible to pay at the moment
    }

    local db = sqlite3.open(self.db_filename)

    for c in db:nrows("SELECT * FROM transactions WHERE creditor = " .. account_id .. " AND revoked = 0;") do
        balance.credits_total = balance.credits_total + c.amount
        if c.payed ~= 0 then balance.credits_payed = balance.credits_payed + c.amount
        else balance.credits_unpayed = balance.credits_unpayed + c.amount end
    end
    
    for d in db:nrows("SELECT * FROM transactions WHERE debitor = " .. account_id .. " AND revoked = 0;")  do
        balance.debits_total = balance.debits_total + d.amount
        if d.payed ~= 0 then balance.debits_payed = balance.debits_payed + d.amount
        else balance.debits_unpayed = balance.debits_unpayed + d.amount end
    end

    balance.balance_total = balance.credits_total - balance.debits_total
    balance.cash = balance.credits_payed - balance.debits_payed
    balance.debt_incurred = balance.debits_unpayed - balance.credits_unpayed - balance.cash

    
    db:close()
    return balance
end


function Bank:getAccount(account_id)
    local db = sqlite3.open(self.db_filename)
    for a in db:nrows("SELECT * FROM accounts WHERE id = " .. account_id .. ";") do 
        db:close()
        a.bank = self
        return a
    end
end

function Bank:statement(account_id)

    local account = self:getAccount(account_id)

    local balance = self:balance(account_id)

    local statement = {
        account = account,
        balance = balance,
        debits = {},
        credits = {},
        transactions = {}
    }

    local db = sqlite3.open(self.db_filename)

    local i = 1
    for t in db:nrows("SELECT * FROM transactions WHERE (creditor = " .. account.id .. " OR debitor = " .. account.id ..") AND revoked = 0 ORDER BY date_created DESC;") do
        statement.transactions[i] = t
        if t.debitor == account.id then
            table.insert(statement.debits, t)
        else
            table.insert(statement.credits, t)
        end
        i = i + 1
    end

    db:close()

    return statement
end


-- The Account

Account = {}

function Account:open(bank, account_id)
    account = bank:getAccount(account_id)

    self.__index = self

    return setmetatable(account, self)
end

function Account:new(bank, holder_name)
    err, account = bank:createAccount(holder_name)

    if err ~= sqlite3.OK then print("Error creating account"); return nil end

    self.__index = self

    return setmetatable(account, self)
end


function Account:orderTransfer(account, amount, concept, due) 
    if amount < 0 then return FLIPIT end -- notify errors
    local date = self.bank:thisDate()
    if due < date then return TIMETRAVEL end
    if account.id == nil then return NEMO end

    local payed = date == due

    local err = self.bank:createTransaction(account.id, self.id, amount, concept, due, payed)
    return err
end

function Account:balance()
    return self.bank:balance(self.id)
end

function Account:print_info()
    local s = self.bank:statement(self.id)

    print(self.holder .. " (" .. self.id .. ") FINANCIAL STATEMENT (DATED " .. self.bank:thisDate() .. ")")
    print("ACCOUNT HOLDER: " .. self.holder)
    print("ACCOUNT ID: " .. self.id)
    print("DATE CREATED: " .. self.date)
    print("-------------------------------------------------------------")
    print("DATE", "DUE", "CONCEPT", "DEBIT", "CREDIT", "ACCOUNT", "PAYED", "ID")

    for _, t in pairs(s.transactions) do
        if t.debitor == self.id then
            print(t.date_created, t.date_due, t.concept, t.amount, "-", t.creditor, t.payed, t.id)
        else
            print(t.date_created, t.date_due, t.concept, "-", t.amount, t.debitor, t.payed, t.id)
        end
    end

    print("-------------------------------------------------------------")

    print(" ", " ", "TOTAL:", s.balance.debits_total, s.balance.credits_total)
    print("BALANCE: " .. s.balance.balance_total, "DEBT: " .. s.balance.debt_incurred)
    print("CASH: " .. s.balance.cash)
end

function help() 
    print("Commands:")
    print("  date               - print the current date")
    print("  next               - advance to the next date")
    print("  revoke             - revoke a transaction")
    print("  create             - create a new account")
    print("  deposit            - make a cash deposit") -- to the vault and the accounts
    print("  withdraw           - make a cash witdrawal") -- from the vault and the account
    print("  balance            - print the balance of an account")
    print("  info               - print the statement of an account")
    print("  exit               - exit the program")
    print("  help               - print this help")
end

function main() 
    print("Enter the Bank name > ")
    local bankname = io.read()
    print("Opening bank... ")
    local bank = Bank:new(bankname)

    help()
    print("> ")
    local cmd = io.read()
    while cmd ~= "exit" do
        if cmd == "date" then
            print("The current date is " .. bank:thisDate())
        elseif cmd == "next" then
            print("Advanced date to " .. bank:nextDate())
        elseif cmd == "revoke" then
            print("Revoke a transaction")
            print("Enter the transaction id > ")
            local transaction_id = io.read("*n")
            bank:revokeTransaction(transaction_id)
        elseif cmd == "create" then
            print("Create a new account")
            print("Enter the account holder name > ")
            local holder_name = io.read()
            local err, account = bank:createAccount(holder_name)
            if err == sqlite3.OK then print("Account created with id " .. account.id) end
        elseif cmd == "deposit" then
            print("Make a deposit")
            print("Enter the account you would like to deposit in > ")
            local account_id = io.read("*n")
            print("Enter the amount to deposit > ")
            local amount = io.read("*n")
            while amount < 0 do 
                print("Deposit must be positive!")
                amount = io.read("*n")
            end
            bank:createTransaction(account_id, -1, amount, "Cash", bank:thisDate(), true)
            bank:createTransaction(0, -1, amount, "["..account_id.."]", bank:thisDate(), true) -- the cash entering the vault
        elseif cmd == "withdraw" then
            print("Make a withdrawal")
            print("Enter the account you would like to withdraw from > ")
            local account_id = io.read("*n")
            print("Enter the amount to withdraw > ")
            local amount = io.read("*n") 
            while amount < 0 do 
                print("Withdrawal must be positive!")
                amount = io.read("*n")
            end
            local balance = bank:balance(account_id)
            local bank_vault = bank:balance(0)

            if balance.cash < amount then print("Not enough cash to withdraw!")
            elseif bank_vault.cash < amount then print("Oops! Corralito!")
            else
                bank:createTransaction(-2, account_id, amount, "Cash", bank:thisDate(), true)
                bank:createTransaction(-2, 0, amount, "["..account_id.."]", bank:thisDate(), true) -- the cash exiting the vault
            end

        elseif cmd == "balance" then
            print("Enter the account id > ")
            local account_id = io.read("*n")
            local balance = bank:balance(account_id)
            print("Account balance:")
            print("CREDITS:")
            print("Total: " .. balance.credits_total)
            print("Payed: " .. balance.credits_payed)
            print("Unpayed: " .. balance.credits_unpayed)
            print("DEBITS:")
            print("Total: " .. balance.debits_total)
            print("Payed: " .. balance.debits_payed)
            print("Unpayed: " .. balance.debits_unpayed)
            print("BALANCE:")
            print("Total: " .. balance.balance_total)
            print("CASH: " .. balance.cash)
            print("DEBT: " .. balance.debt_incurred)
        elseif cmd == "info" then
            print("Enter the account id > ")
            local account_id = io.read("*n")
            local account = Account:open(bank, tonumber(account_id))
            account:print_info()
        elseif cmd == "help" then
            help()
        elseif cmd == "exit" then
            break
        else
            print("Command not recognized")
        end
        cmd = io.read()
    end
end

main()