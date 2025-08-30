# Eco-nomic

*versión en español más abajo*

This is a very simple web application that simulates a very simple banking system.
It is intended to use while playing economic and money-focused board games or similar
(like games over discord or email...), among a group of known people. For example, it would
be suited to play *monopoly* or some sort of *nomic* (check out [Nomic](https://en.wikipedia.org/wiki/Nomic), and 
[Metanopoly](https://johannmg.itch.io/metanopoly)!).

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
[double-entry bookeeping](https://en.wikipedia.org/wiki/Double-entry_bookkeeping). As such the database records transactions on wich one account is
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

---

# Eco-nomic

Esta es una aplicación web muy simple que simula un sistema bancario muy sencillo.
Su propósito es ser utilizada en juegos de mesa o similares enfocados en la economía y el dinero
(como juegos a través de Discord o correo electrónico...), entre un grupo de personas conocidas. Por ejemplo,
sería adecuada para jugar *Monopoly* o algún tipo de *nomic* (¡echa un vistazo a [Nomic](https://en.wikipedia.org/wiki/Nomic) y 
[Metanopoly](https://johannmg.itch.io/metanopoly)!).

## Para qué sirve eco-nomic

- Para una partida de Monopoly con tus amigos si quieres ser un poco *extra*.
- Para un juego con tus amigos de Discord.

## Para qué **NO** sirve eco-nomic

- Para un juego en línea donde esperas que la gente se registre sola. Esto no está implementado intencionalmente.
  La aplicación no ha sido probada en cuanto a seguridad y debes conocer, y preferiblemente confiar en, las personas con las que juegas.

- Para un sistema donde no tengas que gestionar nada. De hecho, tendrás que gestionar la mayoría de las cosas relacionadas
  con el banco en una (sencilla) línea de comandos.

- Para ser una aplicación web elegante. Es extremadamente simple. Sin websockets, solo http. Tienes que
  recargar la página para ver casi todo.

- Para algo serio. Si parece serio, entonces no es para ti; no está hecho para ello.

> NOTA: ESTA NO ES UNA APLICACIÓN SEGURA. NO LA EXPONGAS A INTERNET SI ES POSIBLE. ÚSALA
> DENTRO DE UNA RED LOCAL. SI JUEGAS CON GENTE EN LÍNEA, TAL VEZ USA UNA VPN O ALGO COMO
> HAMACHI.

## Características

Ok, hechas las advertencias y descargos, esto es lo que *eco-nomic* tiene para ofrecer.

- Panel de control de cuenta de usuario simple:
  - Realizar transferencias a otros usuarios.
  - Verificar tus transferencias.
  - Enviar cartas a otros usuarios o al banco.
  - Publicar documentos para que todos los vean.
- Renderizador de documentos con markdown:
  - Soporta extensiones de tipografía (`---` se convierte en una raya larga).
  - Soporta listas de definición.
  - Soporta tachado y otros estilos (estilo de GitHub).
- Modos oscuro y claro.
- Versiones en español e inglés.
- Script Lua para gestionar la base de datos. ¡Modifícalo a tu gusto!

## Instalación

Planeo hacer versiones para Windows y Linux, pero por ahora, tienes que compilarlo tú mismo.
Si tienes Go instalado en tu sistema, debería ser tan fácil como:

    go build

Para la consola Lua, tienes que instalar los paquetes `lsqlite3complete` y `bcrypt`.
Es fácil hacerlo con `luarocks`.

## Usando la aplicación

Primero tienes que generar una base de datos en blanco. Usa el script de Lua para hacerlo. Después de
elegir un idioma, te preguntará si quieres crear una. Simplemente dale un nombre y una contraseña maestra.
Esta contraseña se usará para iniciar sesión en las cuentas especiales del banco (más sobre esto más adelante).

Después de eso, ejecuta el servidor Go así:

    ./eco-nomic <nombre-del-archivo-bd>

La aplicación se servirá en `localhost:8080`. Creo que la aplicación web es bastante intuitiva de usar.

La consola Lua está destinada a que la use el administrador del banco. Es un programa
antiguo, guiado por menús, por lo que no tiene sintaxis que aprender. Actualmente, soporta estos comandos:

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
    imprimir resumen interno del banco

exit:
    exit the program
    salir del programa

help:
    print this help
    imprimir esta ayuda

## Idiosincrasias

Aunque mi intención al construir esto era ser lo más abstracto posible para permitir
muchas formas diferentes de jugar con él, hay algunas cosas codificadas.

### Fechas

La base de datos interna y la aplicación usan y muestran las fechas (dentro del juego) como
números enteros. Esto está destinado a representar la fracción de tiempo más pequeña dentro de tu juego
(tal vez un turno, tal vez una ronda, tal vez una acción). **Tú defines lo que significa**. Pero
ten en cuenta que el sistema procesará la mayoría de las transacciones y eventos como simultáneos
si ocurren en la misma fecha (excepto el envío de cartas y la publicación). Puedes avanzar
en la consola, pero no retroceder (aún). El sistema está concebido para que un paso signifique un turno,
pero no está codificado para ello.

### Las cuentas especiales del banco

El banco tiene tres cuentas propias que están codificadas con los números de cuenta `-2`, `-1` y `0`.
El sistema y la base de datos no registran un saldo como un valor almacenado, sino que se basan en
la [contabilidad por partida doble](https://en.wikipedia.org/wiki/Double-entry_bookkeeping). Como tal, la base de datos
registra transacciones en las que una cuenta es el acreedor y otra el deudor, y calcula
el saldo de esa manera.

Para operar así, el banco tiene una cuenta principal que representa la bóveda (con el número de cuenta `0`).
Este sería el lugar tradicional donde guardas los billetes en Monopoly. Cualquier pago hacia o desde el banco
se realiza a través de esa cuenta. Por otro lado, el número de cuenta `-2` representa la cuenta de *retiros*:
cuando un usuario solicita sacar dinero de su cuenta fuera del banco (lo que sea que eso signifique en tu juego),
se realizará una transacción desde la bóveda a esta cuenta para representar que el banco está entregando el dinero,
y también se pagará una transacción desde la cuenta del jugador a la cuenta de *retiros*,
deduciendo la cantidad de su saldo. La cuenta con el número `-1` es la cuenta de *depósitos*:
esto es lo contrario de la cuenta de *retiros*. Cuando un jugador desea depositar dinero en su cuenta,
se realiza una transacción desde la cuenta de *depósitos* a la bóveda y otra a la cuenta del jugador. Esto (como
en la vida real) crea dinero de forma efectiva: la cuenta de los jugadores solo tiene un registro de transacciones
que suman un valor virtual. El dinero *real* que tiene el banco está asegurado en la bóveda. Esto abre
la posibilidad de que tu juego juegue con dinero fiduciario, un sistema de efectivo y banco, dar
préstamos, etc. También puedes, por supuesto, no darle ninguna importancia.

Las operaciones de retiro y depósito solo se pueden realizar a través del script Lua. El sistema está diseñado
para que el administrador sea también la persona a cargo del banco, como en *Monopoly*, actuando como un cajero del banco.