package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/machinebox/graphql"
	//"encoding/json"
)

var poolsToken0GQL = `
query($token0: String!) {
  token(id: $token0) {
    name
    symbol
    id
    volumeUSD
  }
  pools(
    first: 1000
    orderBy: totalValueLockedUSD
    orderDirection: desc
    where: { token0: $token0 }
  ) {
    id
    volumeUSD
    token0 {
      symbol
    }
    token1 {
      symbol
    }
    
  }
}

`

var poolsToken1GQL = `
query($token1: String!) {
  pools(
    first: 1000
    orderBy: totalValueLockedUSD
    orderDirection: desc
    where: { token1: $token1 }
  ) {
    id
    volumeUSD

    token0 {
      symbol
    }
    token1 {
      symbol
    }
    
  }
}

`

var volumeByAssetIdGQL = `
query($startDate: Int!,$endDate: Int!,$skip: Int!, $poolArray: [String!]!){ poolDayDatas(
  first: 1000
  skip: $skip

  orderBy: id
  orderDirection: desc
  where: { date_gt: $startDate,date_lt:$endDate,pool_in: $poolArray,volumeUSD_gt:0}
) {
  id
  volumeUSD
date
  pool{
    id
    token0{
      symbol
    }
        token1{
      symbol
    }
    
  }
  
}
}

`

var swapsByBlockIdGQL = `
query ($blockNumber: String!){ transactions(
  first: 1000
  orderBy: id
  orderDirection: desc
  where: { blockNumber: $blockNumber,
  }
) {
    id
    blockNumber
  swaps{
    id
    pool{
      id
      liquidity
    }
    token0{
      symbol
    }
    token1{
      symbol
    }
    
  }
  
}
}


`

var swapPairsByBlockIdGQL = `
query ($blockNumber: String!){ transactions(
  first: 1000
  orderBy: id
  orderDirection: desc
  where: { blockNumber: $blockNumber,
  }
) {
    id
  swaps{
    id
    pool{
      id
      liquidity
    }
    token0{
      symbol
    }
    token1{
      symbol
    }
    
  }
  
}
}


`

type dict map[string]interface{}
type PoolList struct {
	Pools []Pool
}

type Pool struct {
	Id        string `json:"id"`
	Key       string `json:"pair"`
	VolumeUSD string `json:"volumeUSD"`
}

func remDupKeys(m PoolList) []Pool {
	keys := make(map[string]bool)
	list := PoolList{}
	for _, entry := range m.Pools {
		if _, ok := keys[entry.Key]; !ok {
			keys[entry.Key] = true
			list.Pools = append(list.Pools, entry)
		}
	}
	return list.Pools
}

func (poollist *PoolList) AddPool(p Pool) []Pool {
	poollist.Pools = append(poollist.Pools, p)

	return poollist.Pools
}

func IsValidAddress(v string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(v)
}

func main() {
	router := gin.Default()
	router.GET("/asset/:assetId/pools", poolsByAssetId)
	router.GET("/asset/:assetId/volume", volumeByAssetId)
	router.GET("/block/:blockId/swaps", swapsByBlockId)
	router.GET("/block/:blockId/swaps/pairs", swapPairsByBlockId)

	router.Run("localhost:8080")
}

func getPools(c *gin.Context, id string, output string) interface{} {

	graphqlClient := graphql.NewClient("https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3")
	graphqlRequest := graphql.NewRequest(poolsToken0GQL)

	graphqlRequest.Var("token0", id)

	var graphqlResponse map[string]interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)
	}

	poollist := PoolList{}

	for _, val := range graphqlResponse["pools"].([]interface{}) {

		id := val.(map[string]interface{})["id"].(string)
		pl := Pool{}

		pl.Id = id
		pl.VolumeUSD = val.(map[string]interface{})["volumeUSD"].(string)
		pl.Key = val.(map[string]interface{})["token0"].(map[string]interface{})["symbol"].(string) + "/" + val.(map[string]interface{})["token1"].(map[string]interface{})["symbol"].(string)

		poollist.AddPool(pl)

	}

	graphqlRequest2 := graphql.NewRequest(poolsToken1GQL)

	graphqlRequest2.Var("token1", id)

	var graphqlResponse2 map[string]interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest2, &graphqlResponse2); err != nil {
		panic(err)
	}

	for _, val := range graphqlResponse2["pools"].([]interface{}) {

		id := val.(map[string]interface{})["id"].(string)

		pl := Pool{}
		pl.Id = id
		pl.VolumeUSD = val.(map[string]interface{})["volumeUSD"].(string)
		pl.Key = val.(map[string]interface{})["token0"].(map[string]interface{})["symbol"].(string) + "/" + val.(map[string]interface{})["token1"].(map[string]interface{})["symbol"].(string)
		poollist.AddPool(pl)

	}

	if output == "full" {
		return remDupKeys(poollist)
	} else {

		var poolids []string

		keys := make(map[string]bool)
		for _, entry := range poollist.Pools {
			if _, ok := keys[entry.Key]; !ok {
				keys[entry.Key] = true
				poolids = append(poolids, entry.Id)
			}
		}
		return poolids

	}
}

func swapPairsByBlockId(c *gin.Context) {

	id := c.Param("blockId")

	graphqlClient := graphql.NewClient("https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3")
	graphqlRequest := graphql.NewRequest(swapPairsByBlockIdGQL)

	graphqlRequest.Var("blockNumber", id)

	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)
	}

	var pairs []string
	pairmap := make(map[int]string)

	for index, val := range graphqlResponse.(map[string]interface{})["transactions"].([]interface{}) {

		swaps := val.(map[string]interface{})["swaps"]
		pairmap[index] = swaps.([]interface{})[0].(map[string]interface{})["token0"].(map[string]interface{})["symbol"].(string) + "/" + swaps.([]interface{})[0].(map[string]interface{})["token1"].(map[string]interface{})["symbol"].(string)
		pairs = append(pairs, swaps.([]interface{})[0].(map[string]interface{})["token0"].(map[string]interface{})["symbol"].(string)+"/"+swaps.([]interface{})[0].(map[string]interface{})["token1"].(map[string]interface{})["symbol"].(string))

	}

	c.IndentedJSON(http.StatusOK, pairs)

}

/*
What is the total volume of that asset swapped in a given time range?
For example, USDC’s Ethereum address and Uniswap ID are both 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.


*/

func volumeByAssetId(c *gin.Context) {
	//var skip int

	id := c.Param("assetId")

	if !IsValidAddress(id) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Invalid Ethereum Address"})
		return
	}

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	if endDate == "" {
		endDate = strconv.FormatInt(time.Now().Unix(), 10)
	}

	skip := c.Query("skip")
	var skipped int

	if _, err := strconv.ParseInt(skip, 10, 64); err == nil {
		skipped, _ = strconv.Atoi(skip)
	} else {
		skipped = 0
	}

	ids := getPools(c, id, "ids")
	joinedIds := ids.([]string)

	graphqlClient := graphql.NewClient("https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3")
	graphqlRequest := graphql.NewRequest(volumeByAssetIdGQL)

	_startDate, _ := strconv.Atoi(startDate)
	_endDate, _ := strconv.Atoi(endDate)

	graphqlRequest.Var("startDate", _startDate)
	graphqlRequest.Var("endDate", _endDate)
	graphqlRequest.Var("poolArray", joinedIds)
	graphqlRequest.Var("skip", skipped)

	var graphqlResponse map[string]interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)
	}

	var totalVolume float64

	for _, val := range graphqlResponse["poolDayDatas"].([]interface{}) {

		fmt.Println(val.(map[string]interface{})["volumeUSD"].(string))
		poolvolume, _ := strconv.ParseFloat(val.(map[string]interface{})["volumeUSD"].(string), 64)
		totalVolume = totalVolume + poolvolume
		//fmt.Println(IsValidAddress(id)) // true
		fmt.Println("ADDED POOLS VOLUME: ", totalVolume)

	}
	fmt.Println("TOTAL POOLS VOLUME: ", totalVolume)
	fmt.Printf("%f\n", totalVolume)

	graphqlResponse["volume"] = totalVolume

	c.IndentedJSON(http.StatusOK, graphqlResponse)

}

/*
Given an asset ID:
What pools exist that include it?
What is the total volume of that asset swapped in a given time range?

For example, USDC’s Ethereum address and Uniswap ID are both 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.

*/

func poolsByAssetId(c *gin.Context) {

	id := c.Param("assetId")

	if !IsValidAddress(id) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": false, "message": "Invalid Ethereum Address"})
		return
	}

	output := getPools(c, id, "full")
	c.IndentedJSON(http.StatusOK, output)

}

func swapsByBlockId(c *gin.Context) {
	id := c.Param("blockId")

	graphqlClient := graphql.NewClient("https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3")
	graphqlRequest := graphql.NewRequest(swapsByBlockIdGQL)
	graphqlRequest.Var("blockNumber", id)
	var graphqlResponse interface{}
	if err := graphqlClient.Run(context.Background(), graphqlRequest, &graphqlResponse); err != nil {
		panic(err)

	}
	c.IndentedJSON(http.StatusOK, graphqlResponse)

}
