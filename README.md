# Picster 
A project to manage pictures and videos

![](https://github.com/wwweidi/picster/workflows/Go/badge.svg)

## Usage
```bash
picster <source folder> <destination folder>
```
will scan the source folder recursively for pictures and videos
and will move these files into the destination folder with an organized structure:
```
destination
    fotos
        yyyy-mm
            yyyymmdd_hhmmss.jpg
        ...
    videos
        yyyy-mm
            yyyymmdd_hhmmss.mov
            yyyymmdd_hhmmss.mp4
            yyyymmdd_hhmmss.avi
        ...
```



## Contributing
Pull requests are welcome!

## License
MIT