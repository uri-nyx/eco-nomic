# Eco-nomic

This is a very simple web application that simulates a very simple banking system.
It is intended to use while playing economic and money-focused board games or similar
(like games over discord or email...), among a group of known people. For example, it would
be suited to play *monopoly* or some sort of *nomic* (check out [Nomic](wikipedia), and 
[Metapoly](itchiolink)!).

## What eco-nomic is for

- A game of monopoly with your friends if you want to be a bit *extra*.
- A game with your discord buds.

## What eco-nomic is **NOT** for

- An online game where you expect new people to register themselves. 
This is not implemented intentionally. The app has not been tested for security, 
and you should know, and preferrably trust, who you're playing with.

- A system where you don't have to manage anything. In fact you'll have to manage 
most bank-related things in a (simple) command line.

- A fancy web app. This is simple in the extreme. No websockets, only http. You have 
to reload the page to see almost everything.

- Anything serious. If it seems serious than play, this is not built for it.

> NOTE: THIS IS NOT A SECURE APP. DO NOT EXPOSE TO THE INTERNET IF POSSIBLE. USE IT 
> WITHIN A LOCAL NETWORK. IF PLAYING WITH PEOPLE ONLINE MAYBE USE A VPN OR SOMETHING LIKE
> HAMACHI.

## Features

Ok, disclaimers and warnings made, this is what *eco-nomic* has to offer.

- Simple user account dashboard:
    - Make transfers to other users.
    - Check your transfers.
    - Send letters to other users or to the bank.
    - Publish documents for everyone to see.
- Document markdown renderer:
    - Supports typography extensions (`---`  becomes an em-dash).
    - Supports definition lists.
    - Supports striketrough and other stylings (github style).
- Dark and light modes.
- Spanish and english versions.
- Lua script to manage the database. Tweak it to your liking!

## Installation

I plan to do releases for windows and linux, but for now, you have to compile yourself.
If you have go installed in your system it should be as easy as:

```sh
go build
```

For the Lua console you have to install the `lsqlite3complete` and `bcrypt` packages. 
It's easy doing it with `luarocks`.

## Using the app

First you have to generate a blank database. Use the lua script to do so. After choosing a language
it will ask you if you want to create one. Simply give it a name and a master password. This password
will be used to log-in on the bank's special accounts (more on that later).

After that run the go server like this:

```sh
./eco-nomic <db-filename>
```

The app will be served at `localhost:8080`. I think the web app is mostly intuitive to use.

The lua console is intended for the bank administrator to use. It is an old timey
menu driven program, so it has no syntax to learn. Currently it supports these commands:
 
```
date: 
    print the current date
    imprimir la fecha actual

next: 
    advance to the next date
    avanzar a la siguiente fecha

revoke: 
    revoke a transaction
    revocar una transacción

create: 
    create a new account
    crear una nueva cuenta

deposit: 
    make a cash deposit
    hacer un depósito de efectivo

withdraw: 
    make a cash withdrawal
    hacer un retiro de efectivo

balance: 
    print the balance of an account
    imprimir el saldo de una cuenta

info: 
    print the statement of an account
    imprimir el estado de una cuenta

accounts: 
    print all the accounts in the bank
    imprimir todas las cuentas en el banco

bank:
    print internal information summary of the bank
    imprimir resumen interno del banco"

exit: 
    exit the program
    salir del programa

help: 
    print this help
    imprimir esta ayuda
```

## Idiosincrasies

While my intention while building this was to be as abstract as possible to allow
many different ways of playing with this, there are some hardcoded things.

### Dates

The internal database and the app use and display the (in-game) dates as integers.
This is meand to represent the smallest fraction of time within your game 
(maybe a turn, maybe a round, maybe an action). **You define what it means**. But take into account
that the system will process most transactions and events as simultaneous if they happen
in the same date (except letter sending and publication). You can advance forward in the 
console but not backwards (yet). The system is conceived so that one step means one turn,
but it's not hardcoded to that.

### The bank special accounts

The bank has three accounts itself that are harcoded to account numbers `-2`, `-1` and `0`.
The system and the database does not track a balance as a stored value, rather, it's based in
double-entry bookeeping. As such the database records transactions on wich one account is
the creditor and another the debitor, and computes balance in that way.

To operate like this, the bank has a primary account that represents the vault (with account number `0`).
This would be the traditional place where you keep the bills in monopoly. Any payments to or from the bank
are made throug that account. On the other hand, account number `-2` represents the *withdrawals* account: when
a user request to get money from their account out of the bank (whatever that means in your game) a transaction will
be performed from the vault to this account to represent the bank giving out the money, and also
a transaction will be payed from the player's account to the *withdrawals* account, deducting the amount
from their balance. The account with number `-1` is the *deposits* account: this is the opposite of the *withdrawals*
account. When a player whishes to deposit some money in their account a transaction is perform from the *deposits* account
to the vault and another to the players account. This (as in real life) effectively creates money: the players account
only holds a record of transactions that amount to a virtual value. The *real* money that the bank has is
secured in the vault. This opens the possibility for your game to play with fiduciary money, a cash and bank system,
give out loans, etc. You can also, of course, not give any thought to it.

Withdrawal and deposit operations can only be performed through the Lua script. The system is designed so that the 
administrator is also the person in charge of the bank, like in *monopoly*, effectively a bank teller.