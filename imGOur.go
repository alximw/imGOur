 
package main

import ("fmt"
	"os"
	"regexp"
	"strings"
	"net/http"
	"path"
	"time"
	"io"
	"io/ioutil"
	"path/filepath"
)

type albuminfo struct{
	
	
	album_type string
	proto string
	album_name string

}

func yellow(text string){

	fmt.Printf("\033[33m %s\033[0m", text)
}

func green(text string){

	fmt.Printf("\033[32m %s\033[0m", text)
}


func red(text string){

	fmt.Printf("\033[31m %s\033[0m", text)
}


func main(){

	args := os.Args
	

	directory := ""
	album := ""

	if (len(args)<2){
		//Wrong number of parameters
		red(fmt.Sprintln("[Error!]:Invalid number of arguments!"))
		usage()
		os.Exit(1)

	}else if (len(args)>2){
		// album link+directory
		directory = args[2]
	}else{
		// just album link
		directory,_ = os.Getwd()
	}
	
	album = args[1]
	
	// get album information
	mAlbumInfo := ParseAlbumInfo(album)
	if mAlbumInfo==nil{
		red(fmt.Sprintf("PARSE ERROR: Couldn't parse album link: %s",album))
		os.Exit(1)
	}
	picture_path := path.Join(directory,mAlbumInfo.album_name)
	
	if (mAlbumInfo.album_type == "S"){
		
		//single picture album (i.e i.imgur.com/whatever)
		yellow(fmt.Sprintf("[+] Downloading picture to directory %s ...",picture_path))
		bytes, time := DownloadLink(album,picture_path)
		green(fmt.Sprintf("Done.Took %d to download %d bytes.",time, bytes))

	}else{

	  	//full sized picture album (more than 1 picture) (i.e imgur.com/a/whatever, imgur.com/wharever, or imgur.com/gallery/whatever)
	  	green(fmt.Sprintf("[+] Downloading album %s to directory: %s \n",album, picture_path))
  	  	//gather all links by scraping the html webpage
  	  	my_links := ParseAlbumWebsite(album,mAlbumInfo)
  	  	
  	  	//count total bytes and miliseconds
  	  	var total_time int64 = 0
  	  	var total_bytes int64 = 0
  	  // for each link in the html...
  	  for key,_ := range my_links{

  	  	sliced_link := strings.Split(key ,"/")
  	  	yellow(fmt.Sprintf("[+] Downloading %s....",key))
  	   	// Download file to the provided path
  	   	bytes, milis := DownloadLink(key, path.Join(picture_path,sliced_link[len(sliced_link)-1]))
  	   	yellow(fmt.Sprintf("DONE\n"))
  	   	// update byte/time count
  	   	total_time += milis
  	   	total_bytes += bytes
  	  }
  	  green(fmt.Sprintf("Finished. Downloaded %d bytes in %d miliseconds\n",total_bytes, total_time))

	  
  	}	

	


}

func ParseAlbumWebsite(album_link string, album_info *albuminfo) map[string]bool{

	// dictionary to be returned. Go doesn't have sets...
	var links map[string]bool = nil
	
	/* For imgur.com/a/whatever request noscript site.
	   this allows us to retrieve all the pictures
	   without actually scrolling the webpage
	*/
	complete_album := album_link
	if (album_info.album_type == "A"){
		complete_album = album_link+"/noscript"
	}

	// Get website html document
	resp, err := http.Get(complete_album)
	if err != nil{
		fmt.Printf("NET Error: Couldn't open link  %s\n",album_link)
	}
	
	if resp.StatusCode != 200{
		fmt.Printf("NET Error: Unexpected http status code\n")
	}
	defer resp.Body.Close()
	
	// Read full response body
	document, err := ioutil.ReadAll(resp.Body)
	if err != nil{
		fmt.Printf("IO Error: Couldn't read response body\n")
		return nil
	}
	
	// Match regex against document and gather all the links
	var re = regexp.MustCompile(`(?m)\{"hash"\:\"(.*?)\".*?\"ext\"\:\"(.*?)\"`)
	results := re.FindAllStringSubmatch(string(document), -1)
	links = make(map[string]bool)
	
	// Add entries to the dictionary and return 

	for _,element := range results{
		link := album_info.proto+"://i.imgur.com/"+element[1]+element[2]
		links[link] = true
	}

	return links
}

func ParseAlbumInfo(album_link string) *albuminfo {
	

	proto := ""
	hostname := ""
	var info *albuminfo = nil
	
	re,_ := regexp.Compile(`(?m)(https?):\/{2}(.*?)\/`)
	results := re.FindAllStringSubmatch(album_link, -1)

	// Check album type

	if (len(results[0])<3){
		fmt.Println("Error. Link format is not correct")
		return nil
	
	}else{
		proto = results[0][1]
		hostname = results[0][2]
		// Direct link (single image album)
		if (hostname=="i.imgur.com"){
			//single pic album
			sliced_url := strings.Split(album_link ,"/")
			filename := sliced_url[len(sliced_url)-1]
			info = &albuminfo{album_type:"S",proto:proto,album_name:filename}
			return info
		
		// imgur.com link		
		}else if(hostname=="imgur.com" ||hostname=="m.imgur.com"){
			// multiple pic album
			// Regex with ?groups to find out the album type
			re,_ := regexp.Compile(`(?m)(https?):\/\/.*?\/(a\/|gallery\/)?(.*)`)	
			results := re.FindAllStringSubmatch(album_link, -1)
			
			if results[0][2]=="a/"{
				// album
				info = &albuminfo{album_type:"A",proto:proto,album_name:results[0][len(results[0])-1]}
			}else if results[0][2]==""{
				//Single-pic album
				info = &albuminfo{album_type:"SA",proto:proto,album_name:results[0][len(results[0])-1]}
			}else if results[0][2]=="gallery/"{
				//gallery album
				info = &albuminfo{album_type:"G",proto:proto,album_name:results[0][len(results[0])-1]}
			}else{
				return nil
			}
			return info
			
		}else{
			red("Error. Not a valid Imgur.com link")
		}
	}
	return nil

}

func DownloadLink(link string, path string)(int64, int64){
	
	var downloaded_bytes int64 = 0
	var elapsed_milis int64 = 0

	start := CurrentMilis()
	
	// Get file parent directory
	directory := filepath.Dir(path)

	// Create parent directory if it doesn't exist
	err := os.MkdirAll(directory, 0755)

	//Create file
	out_file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		red(fmt.Sprintf("IO Error: Couldn't create file %s\n",path))
		return 0,0
	}
	defer out_file.Close()

	// Open picture link
	resp, err := http.Get(link)
	if err != nil{
		red(fmt.Sprintf("NET Error: Couldn't open link  %s\n",link))
		return 0,0
	}

	if resp.StatusCode != 200{
		red(fmt.Sprintf("NET Error: Unexpected http status code\n"))
		return 0,0
	}
	defer resp.Body.Close()

	// Copy response body to file

	bytes,err := io.Copy(out_file, resp.Body)
	if err != nil{
		red(fmt.Sprintf("IO Error: Couln't copy response body to file"))
		return 0,0
	}
	// update time and bytes counter
	downloaded_bytes = bytes
	end := CurrentMilis()
	elapsed_milis = end-start

	return downloaded_bytes, elapsed_milis

	}

func CurrentMilis() int64 {
    return time.Now().UnixNano() / int64(time.Millisecond)
}

func usage(){
	green(fmt.Sprintf("USAGE: imgour <imgour link> [directory]\n"))
}

