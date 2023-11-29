package main

type RequestDatum struct {
	p        Peer
	path     string
	hash     [32]byte
	type_req byte // 0 -> List les nom seulement, 1 -> download pour de vrai
}

var reqDatum []RequestDatum = make([]RequestDatum, 0)
