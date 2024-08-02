an attempt to reproduce a distributed lockup in groupcache. essentially, if peer lists disagree between peers, it's possible for peer a to belive peer b is the owner of an asset, while peer b believes peer a is the owner. they will wait on each other indefinitely. peer lists will disagree for a moment, however short, during rolling deployments, when a node is removed or added, etc.

1. `go run main.go --port=8080 --peers=http://localhost:8081,http://localhost:8081,http://localhost:8082`
1. `go run main.go --port=8081 --peers=http://localhost:8081,http://localhost:8082`
1. `go run main.go --port=8082 --peers=http://localhost:8081,http://localhost:8081`
1. `curl localhost:8080/things/foobar` (note that this hangs indefinitely)
1. repeat curl in new terminal (leaving old one running) and note it still hangs indefinitely
1. cancel first curl
1. note that the second curl automatically cancels
1. get the curl stuck again
1. fix node 1's peer list: `curl -d 'http://localhost:8080,http://localhost:8081,http://localhost:8082' localhost:8080/peers`
1. notice curl is still stuck; cancel again
1. curl once more, notice it returns now