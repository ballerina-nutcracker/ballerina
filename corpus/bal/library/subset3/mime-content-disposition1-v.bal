import ballerina/io;
import ballerina/mime;

public function main() {
    mime:ContentDisposition cd = mime:getContentDispositionObject("form-data; name=\"file\"; filename=\"test.txt\"");
    io:println(cd.disposition);
    io:println(cd.name);
    io:println(cd.fileName);

    mime:ContentDisposition newCd = new ();
    newCd.disposition = "attachment";
    newCd.fileName = "report.pdf";
    io:println(newCd.toString());

    mime:ContentDisposition quotedCd = new ();
    quotedCd.disposition = "attachment";
    quotedCd.fileName = "my report.pdf";
    io:println(quotedCd.toString());
}
// @output form-data
// @output file
// @output test.txt
// @output attachment; filename=report.pdf
// @output attachment; filename="my report.pdf"
