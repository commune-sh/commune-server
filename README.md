#### Commune

Commune lets you build free and open public communities on a matrix server. It transforms a Matrix server into a publicly accessible community platform. The goal is to make it easy for existing homeserver operators to open up their Matrix instance (or a subset of it) to the web, and add extra community features using native Matrix functionality.

To see a live running instance, check out [Shpong](https://shpong.com)

#### What does it do?
Commune opens up all spaces and underlying rooms to the web by reading data directly from the Synapse DB, bypassing Synapse's client-server API. Additional features such as discussion boards, threaded comments are rendered by the [client](https://github.com/commune-os/commune-client). Commune makes use of many [materialized views](https://github.com/commune-os/commune-server/tree/main/db/matrix/views) for querying Synapse events.

#### How to run
You'll need to set up a matrix/synapse server. Existing servers can be used too, but Commune is highly experimental at the moment, so it's best to set up a new homeserver.

Check back here soon for instructions.

##### WARNING
Commune is operating in `world_readable` mode. This means that everything on your matrix server has the potential to be accessible from the web. No work has been put into private spaces/rooms or encryption. Unless explicitly stated, assume that every event on a Commune-based matrix server will be public.
