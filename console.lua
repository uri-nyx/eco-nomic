-- Simple bank account manager for transactions in games

--[[

Accounts are the base of the system.
They are identified by a unique integer and tied to a holder.
The account does not have a balance, but tracks a list of transactions.
This transactions can be credits, or debits.

--]]

MAX_TRANSACTIONS = 99999 -- arbitrary number
ACCOUNT_MIN = 1000
ACCOUNT_MAX = 9999
MAX_AMOUNT = 999999999 -- arbitrary number
ERR = { NOID = 1000, TIMETRAVEL = 1001, NEGATIVE = 1002 }

Lang = "en"
local languages = { "en", "es" }
local msg = { error = {} }

msg.bank_open_or_create = {
    ["en"] = "Do you want to open an existing bank or create a new one?",
    ["es"] = "¿Quieres abrir un banco existente o crear uno nuevo?"
}
msg.bank_open = {
    ["en"] = "Open an existing bank",
    ["es"] = "Abrir un banco existente"
}
msg.bank_create = {
    ["en"] = "Create a new bank",
    ["es"] = "Crear un banco nuevo"
}
msg.invalid_input = {
    ["en"] = "That's not a valid option. Please try again.",
    ["es"] = "Esa no es una opción válida. Por favor, inténtalo de nuevo."
}
msg.select_lang =
"Select your language\nSelecciona tu idioma"
msg.invalid_lang =
"That language is not supported. Please choose from the list.\nEse idioma no es compatible. Por favor, elige de la lista."

msg.input_name = {
    ["en"] = "Enter bank name",
    ["es"] = "Introduce el nombre del banco"
}
msg.input_password = {
    ["en"] = "Enter master password",
    ["es"] = "Introduce la contraseña maestra"
}
msg.withdrawals = {
    ["en"] = "WITHDRAWALS",
    ["es"] = "RETIRADAS"
}
msg.deposits = {
    ["en"] = "DEPOSITS",
    ["es"] = "DEPOSITOS"
}
msg.bank_vault = {
    ["en"] = "VAULT",
    ["es"] = "CAJA"
}
msg.not_a_valid_command = {
    ["en"] = "Not a valid command. Type 'help' to see available commands.",
    ["es"] = "Comando no válido. Escribe 'help' para ver los comandos disponibles."
}
msg.error = {
    open_bank = {
        ["en"] = "Database file not found or could not be opened.",
        ["es"] = "Archivo de base de datos no encontrado o no se pudo abrir."
    },
    create_bank = {
        ["en"] = "Could not create database.",
        ["es"] = "No se pudo crear la base de datos."
    },
    unreachable = {
        ["en"] = "An internal error occurred. This is a bug.",
        ["es"] = "Ocurrió un error interno. Esto es un fallo del sistema."
    },
    internal = {
        ["en"] = "An internal error occurred.",
        ["es"] = "Ocurrió un error interno."
    },

    account_not_found = {
        ["en"] = "Account ID not found in the database.",
        ["es"] = "El ID de cuenta no se encontró en la base de datos"
    }
}
msg.cannot_create = {
    ["en"] = "Failed to create the account.",
    ["es"] = "No se pudo crear la cuenta."
}
msg.not_enough_money = {
    ["en"] = "You don't have enough cash in your account.",
    ["es"] = "No tienes suficiente efectivo en tu cuenta."
}
msg.bank_has_no_money = {
    ["en"] = "The bank vault does not have enough cash for this withdrawal.",
    ["es"] = "La caja fuerte del banco no tiene suficiente efectivo para este retiro."
}
msg.current_date = {
    ["en"] = "Current date: ",
    ["es"] = "Fecha actual: "
}
msg.advanced_date = {
    ["en"] = "Date advanced to: ",
    ["es"] = "Fecha avanzada a: "
}
msg.revoke_transaction = {
    ["en"] = "Enter the transaction ID to revoke",
    ["es"] = "Introduce el ID de la transacción a revocar"
}
msg.enter_account_holder = {
    ["en"] = "Enter the new account holder's name",
    ["es"] = "Introduce el nombre del nuevo titular de la cuenta"
}
msg.enter_password = {
    ["en"] = "Enter the account password",
    ["es"] = "Introduce la contraseña de la cuenta"
}
msg.account_new_id = {
    ["en"] = "New account successfully created with ID ",
    ["es"] = "Nueva cuenta creada con éxito, con el ID "
}
msg.enter_account_id = {
    ["en"] = "Enter the account ID",
    ["es"] = "Introduce el ID de la cuenta"
}
msg.enter_amount = {
    ["en"] = "Enter the amount",
    ["es"] = "Introduce la cantidad"
}
msg.cash = {
    ["en"] = "CASH",
    ["es"] = "EFECTIVO"
}
msg.command = {
    ["en"] = "Command",
    ["es"] = "Comando"
}


-- Accounts and transactions are stored in a local sqlite database
local sqlite3 = require("lsqlite3complete")

-- Passwords are hashed with the bcrypt algorithm
local bcrypt = require("bcrypt")

-- Database
local tables = [[
    CREATE TABLE IF NOT EXISTS system (
        id INTEGER NOT NULL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        clock INTEGER
    );

    CREATE TABLE IF NOT EXISTS accounts (
        id INTEGER NOT NULL PRIMARY KEY,
        holder VARCHAR(100) NOT NULL,
        date INTEGER NOT NULL,
        password TEXT NOT NULL
    );

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
        public BOOLEAN,
        FOREIGN KEY (sender) REFERENCES accounts (id),
        FOREIGN KEY (receiver) REFERENCES accounts (id)
    );
]]

-- This accounts are used internaly by the bank when giving out loans, or when
-- users make deposits or witdrawals
local reserved_accounts = {
    [[ INSERT INTO system (name, clock)
    VALUES($1, 0); ]],

    [[ INSERT INTO accounts (id, holder, date, password)
    VALUES(-2, $2, 0, $5); ]],

    [[ INSERT INTO accounts (id, holder, date, password)
    VALUES(-1, $3, 0, $5); ]],

    [[ INSERT INTO accounts (id, holder, date, password)
    VALUES(0, $4, 0, $5); ]]
}

Bank = {}

local function file_exists(name)
    local f = io.open(name, "r")
    return f ~= nil and io.close(f)
end

function Bank:open(bank_name)
    if not file_exists(bank_name .. ".sqlite3") then return nil end

    local db, _, err = sqlite3.open(bank_name .. ".sqlite3")
    if db == nil then return nil end

    if db:close() ~= sqlite3.OK then return nil end

    local bank = {
        name = bank_name,
        db_filename = bank_name .. ".sqlite3",
        db = db,
        error = nil
    }

    self.__index = self

    return setmetatable(bank, self), nil
end

function Bank:new(bank_name, withdrawals, deposits, vault, master_password)
    if file_exists(bank_name .. ".sqlite3") then
        return nil
    end

    local db, _, err = sqlite3.open(bank_name .. ".sqlite3")
    if db == nil then
        return nil
    end

    local err = db:exec(tables)
    if err ~= sqlite3.OK then
        return nil
    end


    local system_reserved = db:prepare(reserved_accounts[1])
    if system_reserved == nil then
        db:close()
        return nil
    end

    err = system_reserved:bind_values(bank_name)
    if err ~= sqlite3.OK then
        return nil
    end

    err = system_reserved:step()
    if err ~= sqlite3.DONE then
        return nil
    end

    local hash = bcrypt.digest(master_password, 10)
    local names = { withdrawals, deposits, vault }

    for i = 2, #reserved_accounts do
        local reserved = db:prepare(reserved_accounts[i])
        if reserved == nil then
            db:close()
            return nil
        end

        err = reserved:bind_values(names[i - 1], hash)
        if err ~= sqlite3.OK then
            return nil
        end

        err = reserved:step()
        if err ~= sqlite3.DONE then
            return nil
        end
    end

    err = db:close()
    if err ~= sqlite3.OK then
        return nil
    end

    local bank = {
        name = bank_name,
        db_filename = bank_name .. ".sqlite3",
        db = db,
    }

    self.__index = self

    return setmetatable(bank, self), nil
end

function Bank:reopen()
    local db, code, _ = sqlite3.open(self.db_filename)
    if db == nil then
        return code
    end

    self.db = db
    return nil
end

function Bank:close()
    local err = self.db:close()
    if err ~= sqlite3.OK then
        return err
    else
        return nil
    end
end

function Bank:insert(table_name, fields, values)
    local err = self:reopen()
    if err ~= nil then return err end

    err = self.db:exec("INSERT INTO " .. table_name .. " (" .. fields .. ") VALUES(" .. values .. ");")
    if err ~= sqlite3.OK then return err end

    return self:close()
end

function Bank:update(table_name, field, value1, where, value2)
    local err = self:reopen()
    if err ~= nil then return err end

    err = self.db:exec("UPDATE " ..
        table_name .. " SET " .. field .. " = " .. value1 .. " WHERE " .. where .. " = " .. value2 .. ";")

    if err ~= sqlite3.OK then return err end

    return self:close()
end

function Bank:thisDate()
    local err = self:reopen()
    if err ~= nil then return nil, err end

    for d in self.db:nrows("SELECT clock FROM system WHERE id = 1;") do
        err = self:close()
        return d.clock, err
    end
end

function Bank:nextDate()
    local date, err = self:thisDate()
    if date == nil then return nil, err end

    local err = self:reopen()
    if err ~= nil then return nil, err end

    err = self.db:exec("UPDATE system SET clock =" .. date + 1 .. " WHERE id = 1;"
        .. "UPDATE transactions SET payed = 1 WHERE date_due >= (SELECT clock FROM system WHERE id = 1);")
    if err ~= sqlite3.OK then return nil, err end

    err = self:close()
    return date + 1, err
end

function Bank:createTransaction(creditor_id, debitor_id, amount, concept, due, payed)
    -- recheck error handling methods
    if creditor_id == nil or debitor_id == nil then return ERR.NOID end

    local date, err = self:thisDate()
    if date == nil then return err end

    if due < date then return ERR.TIMETRAVEL end
    if amount < 0 then return ERR.NEGATIVE end

    err = self:insert(
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

function Bank:createAccount(holder_name, password)
    if holder_name == nil then return nil end
    if password == nil then return nil end

    local date, err = self:thisDate()
    if err ~= nil then return nil end

    local hash = bcrypt.digest(password, 10)

    local err = self:reopen()
    if err ~= nil then return nil end

    -- repeat loop
    repeat
        local sql =
            "INSERT INTO accounts (id, holder, date, password)" ..
            "VALUES(" ..
            math.random(ACCOUNT_MIN, ACCOUNT_MAX) .. ",\"" .. holder_name .. "\"," .. date .. ",\"" .. hash .. "\");"
        err = self.db:exec(sql)
    until err ~= sqlite3.CONSTRAINT

    if err ~= sqlite3.OK then return nil end

    local id = self.db:last_insert_rowid() -- would be 0 if exec didn't succeed but that is unreachable
    if id == 0 then return nil end


    local account = {
        id = id,
        holder = holder_name,
        date = date,
        bank = self
    }

    if self:close() then
        return nil
    else
        return account
    end
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

    local err = self:reopen()
    if err ~= nil then return nil end

    for c in self.db:nrows("SELECT * FROM transactions WHERE creditor = " .. account_id .. " AND revoked = 0;") do
        balance.credits_total = balance.credits_total + c.amount
        if c.payed ~= 0 then
            balance.credits_payed = balance.credits_payed + c.amount
        else
            balance.credits_unpayed = balance.credits_unpayed + c.amount
        end
    end

    for d in self.db:nrows("SELECT * FROM transactions WHERE debitor = " .. account_id .. " AND revoked = 0;") do
        balance.debits_total = balance.debits_total + d.amount
        if d.payed ~= 0 then
            balance.debits_payed = balance.debits_payed + d.amount
        else
            balance.debits_unpayed = balance.debits_unpayed + d.amount
        end
    end

    balance.balance_total = balance.credits_total - balance.debits_total
    balance.cash = balance.credits_payed - balance.debits_payed
    balance.debt_incurred = balance.debits_unpayed - balance.credits_unpayed - balance.cash


    if self:close() then
        return nil
    else
        return balance
    end
end

function Bank:getAccount(account_id)
    local err = self:reopen()
    if err ~= nil then
        print("reopen")
        return nil
    end
    for a in self.db:nrows("SELECT * FROM accounts WHERE id = " .. account_id .. ";") do
        if self:close() then
            print("close")
            return nil
        else
            a.bank = self
            return a
        end
    end
end

function Bank:listAccounts()
    local err = self:reopen()
    if err ~= nil then return nil end
    for a in self.db:nrows("SELECT * FROM accounts;") do
        print(a.holder, a.id) --TODO: return nice table string
    end
    return self:close()
end

function Bank:statement(account_id)
    local account = self:getAccount(account_id)
    if account == nil then return nil end

    local balance = self:balance(account_id)
    if balance == nil then return nil end

    local statement = {
        account = account,
        balance = balance,
        debits = {},
        credits = {},
        transactions = {}
    }

    local err = self:reopen()
    if err ~= nil then return nil end

    local i = 1
    for t in self.db:nrows("SELECT * FROM transactions WHERE (creditor = " .. account.id .. " OR debitor = " .. account.id .. ") AND revoked = 0 ORDER BY date_created DESC;") do
        statement.transactions[i] = t
        if t.debitor == account.id then
            table.insert(statement.debits, t)
        else
            table.insert(statement.credits, t)
        end
        i = i + 1
    end

    if self:close() then
        return nil
    else
        return statement
    end
end

-- The Account

Account = {}

function Account:open(bank, account_id)
    local account = bank:getAccount(account_id)
    if account == nil then return nil end

    self.__index = self

    return setmetatable(account, self)
end

function Account:new(bank, holder_name, password)
    local account = bank:createAccount(holder_name, password)
    if account == nil then return nil end

    self.__index = self

    return setmetatable(account, self)
end

function Account:orderTransfer(account, amount, concept, due)
    if amount < 0 then return ERR.NEGATIVE end -- notify errors
    local date, err = self.bank:thisDate()
    if err ~= nil then return err end

    if due < date then return ERR.TIMETRAVEL end

    if account.id == nil then return ERR.NOID end

    local payed = date == due

    return self.bank:createTransaction(account.id, self.id, amount, concept, due, payed)
end

function Account:balance()
    return self.bank:balance(self.id)
end

function Account:print_info()
    -- TODO: return a tabulated string
    local s = self.bank:statement(self.id)
    if s == nil then return nil end

    local date, err = self.bank:thisDate()
    if err ~= nil then return nil end

    print(self.holder .. " (" .. self.id .. ") FINANCIAL STATEMENT (DATED " .. date .. ")")
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

-- Command line interface

local commands = {
    ["date"] = {
        ["en"] = "print the current date",
        ["es"] = "imprimir la fecha actual"
    },
    ["next"] = {
        ["en"] = "advance to the next date",
        ["es"] = "avanzar a la siguiente fecha"
    },
    ["revoke"] = {
        ["en"] = "revoke a transaction",
        ["es"] = "revocar una transacción"
    },
    ["create"] = {
        ["en"] = "create a new account",
        ["es"] = "crear una nueva cuenta"
    },
    ["deposit"] = {
        ["en"] = "make a cash deposit",
        ["es"] = "hacer un depósito de efectivo"
    },
    ["withdraw"] = {
        ["en"] = "make a cash withdrawal",
        ["es"] = "hacer un retiro de efectivo"
    },
    ["balance"] = {
        ["en"] = "print the balance of an account",
        ["es"] = "imprimir el saldo de una cuenta"
    },
    ["info"] = {
        ["en"] = "print the statement of an account",
        ["es"] = "imprimir el estado de una cuenta"
    },
    ["accounts"] = {
        ["en"] = "print all the accounts in the bank",
        ["es"] = "imprimir todas las cuentas en el banco"
    },
    ["bank-info"] = {
        ["en"] = "print internal information summary of the bank",
        ["es"] = "imprimir resumen interno del banco",
    },
    ["exit"] = {
        ["en"] = "exit the program",
        ["es"] = "salir del programa"
    },
    ["help"] = {
        ["en"] = "print this help",
        ["es"] = "imprimir esta ayuda"
    }
}

local function help()
    print("Commands:")
    for command, descriptions in pairs(commands) do
        if descriptions[Lang] then
            print(string.format("    %-12s - %s", command, descriptions[Lang]))
        else
            print(string.format("    %-12s - %s", command, descriptions["en"]))
        end
    end
end

local function prompt(s)
    io.write(s)
    io.write(" > ")
    io.flush()
end

local function getn()
    local n
    repeat
        n = io.read("*n")
        _ = io.read("*l")
    until n ~= nil
    return n
end

local function input_number(s, min, max)
    local n
    local times = 0
    prompt(s)

    repeat
        if times > 0 then
            print(msg.invalid_input[Lang])
            prompt(s)
        end

        n = getn()
        times = times + 1
    until n >= min and n <= max

    return n
end

local function input_s(s)
    prompt(s)
    return io.read()
end

local function panic(s)
    print("FATAL: ", s)
    os.exit(1)
end

local function select_language()
    print(msg.select_lang)
    for k, i in ipairs(languages) do
        print(k .. ": " .. i)
    end

    local i = getn()
    while i > #languages or i < 1 do
        print(msg.invalid_lang)
        i = getn()
    end

    return languages[i]
end

local function open_or_create_bank()
    print(msg.bank_open_or_create[Lang])

    print("1: " .. msg.bank_open[Lang])
    print("2: " .. msg.bank_create[Lang])

    local i = input_number("", 1, #languages)

    if i == 1 then
        local name = input_s(msg.input_name[Lang])
        local bank = Bank:open(name)
        if bank == nil then
            panic(msg.error.open_bank[Lang])
        else
            return bank
        end
    elseif i == 2 then
        local name = input_s(msg.input_name[Lang])
        local pass = input_s(msg.input_password[Lang])
        local w = msg.withdrawals[Lang]
        local d = msg.deposits[Lang]
        local b = msg.bank_vault[Lang]

        local bank = Bank:new(name, w, d, b, pass)
        if bank == nil then
            panic(msg.error.create_bank[Lang])
        else
            return bank
        end
    else
        panic(msg.error.unreachable[Lang])
    end
end

local function internal_error()
    print(msg.error.internal[Lang])
end


local function main()
    Lang = select_language()
    local bank = open_or_create_bank()

    help()

    repeat
        local cmd = input_s(msg.command[Lang])

        if cmd == "date" then
            local date, err = bank:thisDate()
            if err ~= nil then
                internal_error()
            else
                print(msg.current_date[Lang] .. date)
            end
        elseif cmd == "next" then
            local date, err = bank:nextDate()
            if err ~= nil then
                internal_error()
            else
                print(msg.advanced_date[Lang] .. date)
            end
        elseif cmd == "revoke" then
            local transaction_id = input_number(msg.revoke_transaction[Lang], 0, MAX_TRANSACTIONS)
            if bank:revokeTransaction(transaction_id) then internal_error() end
        elseif cmd == "create" then
            local holder_name = input_s(msg.enter_account_holder[Lang])
            local password = input_s(msg.enter_password[Lang])
            local account = bank:createAccount(holder_name, password)
            if account == nil then
                internal_error()
            else
                print(msg.account_new_id[Lang] .. account.id)
            end
        elseif cmd == "deposit" then
            local account_id = input_number(msg.enter_account_id[Lang], ACCOUNT_MIN, ACCOUNT_MAX)
            local amount = input_number(msg.enter_amount[Lang], 0, MAX_AMOUNT)
            if bank:createTransaction(account_id, -1, amount, msg.cash[Lang], bank:thisDate(), true) then internal_error() end
            if bank:createTransaction(0, -1, amount, "[" .. account_id .. "]", bank:thisDate(), true) then internal_error() end
        elseif cmd == "withdraw" then
            local account_id = input_number(msg.enter_account_id[Lang], ACCOUNT_MIN, ACCOUNT_MAX)
            local amount = input_number(msg.enter_amount[Lang], 0, MAX_AMOUNT)

            local balance = bank:balance(account_id)
            local bank_vault = bank:balance(0)

            if balance == nil or bank_vault == nil then
                internal_error()
            else
                if balance.cash < amount then
                    print(msg.not_enough_money[Lang])
                elseif bank_vault.cash < amount then
                    print(msg.bank_has_no_money[Lang])
                else
                    if bank:createTransaction(-2, account_id, amount, msg.cash[Lang], bank:thisDate(), true) then
                        internal_error()
                    end
                    if bank:createTransaction(-2, 0, amount, "[" .. account_id .. "]", bank:thisDate(), true) then
                        internal_error()
                    end
                end
            end
        elseif cmd == "balance" then
            -- TODO accurate balance printing
            local account_id = input_number(msg.enter_account_id[Lang], ACCOUNT_MIN, ACCOUNT_MAX)
            local balance = bank:balance(account_id)
            if balance == nil then
                internal_error()
            else
                print("CASH: ", balance.balance_total)
            end
        elseif cmd == "info" then
            local account_id = input_number(msg.enter_account_id[Lang], ACCOUNT_MIN, ACCOUNT_MAX)
            local account = Account:open(bank, tonumber(account_id))
            if account == nil then
                print(msg.error.account_not_found[Lang])
            else
                account:print_info()
            end
        elseif cmd == "bank-info" then
            local account = Account:open(bank, 0)
            if account == nil then internal_error() else account:print_info() end
            account = Account:open(bank, -1)
            if account == nil then internal_error() else account:print_info() end
            account = Account:open(bank, -2)
            if account == nil then internal_error() else account:print_info() end
        elseif cmd == "accounts" then
            bank:listAccounts()
        elseif cmd == "help" then
            help()
        elseif cmd == "exit" then
            break
        else
            print(msg.not_a_valid_command[Lang])
        end
    until cmd == "exit"
end

main()
