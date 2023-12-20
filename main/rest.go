package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fatih/color"
)

const server string = "https://jch.irif.fr:8443/"
const server_name_peer string = "jch.irif.fr"

var debug_rest bool = false

func getRequest(c *http.Client, URL string) (*http.Response, error) {
	if debug_rest {
		fmt.Println("[getRequest] GET at addr:", URL)
	}

	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(req)

	if debug_rest {
		fmt.Println("[getRequest] GET done")
	}

	return res, err
}

func GetPeers(c *http.Client) ([]string, error) {
	if debug_rest {
		fmt.Println("[GetPeers] Calling getRequest")
	}

	res, err := getRequest(c, server+"peers/")
	if err != nil {
		return nil, err
	}

	// * We're gonna store the peers we got in this array
	arr_peers := make([]string, 0)

	if res.StatusCode == 200 {
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			content := scanner.Text()
			if content != "" {
				arr_peers = append(arr_peers, content)
			}
		}

		if debug_rest {
			fmt.Println("[GetPeers] found peers:", arr_peers)
		}

		return arr_peers, nil
	}

	if debug_rest {
		color.Red("[GetPeers] Invalid status code\n")
	}

	return nil, errors.New("invalid status code")
}

func GetAddresses(c *http.Client, peer string) ([]string, error) {
	if debug_rest {
		fmt.Println("[GetAddresses] Calling getRequest")
	}

	res, err := getRequest(c, server+"peers/"+peer+"/addresses")
	if err != nil {
		return nil, err
	}

	// * We're gonna store the addresses we got in this array
	arr_addr := make([]string, 0)

	if res.StatusCode == 200 {
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			content := scanner.Text()
			if content != "" {
				arr_addr = append(arr_addr, content)
			}
		}

		if debug_rest {
			fmt.Println("[GetAddresses] found addresses:", arr_addr)
		}

		return arr_addr, nil
	}

	if res.StatusCode == 404 {
		if debug_rest {
			color.Magenta("[GetAddresses] Unknown Peer\n")
		}
		return nil, errors.New("unknown peer")
	}

	if debug_rest {
		color.Red("[GetAddresses] Invalid status code\n")
	}
	return nil, errors.New("invalid status code")
}

func GetKey(c *http.Client, peer string) ([]byte, error) {
	if debug_rest {
		fmt.Println("[GetKey] Calling getRequest")
	}

	res, err := getRequest(c, server+"peers/"+peer+"/key")
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		key, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if debug_rest {
			fmt.Printf("[GetKey] found key: %x\n", key)
		}
		return key, nil
	}
	if res.StatusCode == 204 {
		if debug_rest {
			fmt.Println("[GetKey] No key registered")
		}
		return nil, nil
	}
	if res.StatusCode == 404 {
		if debug_rest {
			color.Magenta("[GetKey] Unknown peer\n")
		}
		return nil, errors.New("unknown peer")
	}

	if debug_rest {
		color.Red("[GetKey] Invalid status code\n")
	}
	return nil, errors.New("invalid status code")
}

func GetRoot(c *http.Client, peer string) ([]byte, error) {
	if debug_rest {
		fmt.Println("[GetRoot] Calling getRequest")
	}

	res, err := getRequest(c, server+"peers/"+peer+"/root")
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		root, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if debug_rest {
			fmt.Printf("[GetRoot] found root: %x\n", root)
		}
		return root, nil
	}
	if res.StatusCode == 204 {
		if debug_rest {
			fmt.Println("[GetRoot] No root registered")
		}
		return nil, nil
	}
	if res.StatusCode == 404 {
		if debug_rest {
			color.Magenta("[GetRoot] Unknown peer\n")
		}
		return nil, errors.New("unknown peer")
	}

	if debug_rest {
		color.Red("[GetRoot] Invalid status code\n")
	}
	return nil, errors.New("invalid status code")
}
