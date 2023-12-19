function export_file(data_file) {
    const utf8EncodeText = new TextEncoder();
    let byte_file =  Array.from(utf8EncodeText.encode(data_file));
    let buff = [];
    for(i = 0; i < byte_file.length; i += 1024) {
        chunk = byte_file.slice(i*1024, (i+1)*1024);
        console.log(chunk);
        chunk.unshift(0);

        console.log(chunk);


    }



}