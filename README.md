# nasa - Go library and CLI for NASA API

### Basic Library Usage
``` go
package main

import "github.com/peteretelej/nasa"

func main(){
	apod,err:= nasa.ApodImage(time.Now())
	handle(err)
	fmt.Println(apod)

	// apod has structure of nasa.Image, hence get details with:
	// apod.Date, apod.Title, apod.Explanation, apod.URL, apod.HDURL etc
	fmt.Printf("Today's APOD is %s, available at %s",apod.Title,apod.HDURL)

	lastweek := time.Now().Add(-(7*24*time.Hour))
	apod,err= nasa.ApodImage(lastweek)
	handle(err)
	fmt.Printf("APOD for 1 week ago:\n%s\n",apod)
}
func handle(err error){
	if err!=nil{
		log.Fatal(err)
	}
}
```


## nasa CLI

``` sh
go get -u github.com/peteretelej/nasa/cmd/nasa

nasa apod 
# returns the NASA Astronomy Picture of the day

nasa apod -date 2016-01-17 
# returns the NASA APOD for the date specified
```
