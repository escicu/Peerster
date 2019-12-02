
setInterval(update,1000);

$.urlParam = function(name){
	var results = new RegExp('[\?&]' + name + '=([^&#]*)').exec(window.location.href);
	return results[1] || 0;
};


function update() {
var obj={
				dest:$.urlParam("dest")
			};
var query=$.ajax({
	url : '/private',
	data : obj

});

query.done(function( data ) {
		var chatbox=document.getElementById("chat");
		chatbox.innerHTML="";

		for (m in data) {
			chatbox.innerHTML+="<strong>"+data[m]["Origin"]+"</strong>: "+data[m]["Text"]+"<br />";
			}
		chatbox.scrollTop(chatbox.height());
});
query.fail(function( ) {
		document.getElementById("chat").innerHTML+="Connection problem<br />";
 });

}


$(document).ready(function(){

  $('input[name=send]').click(function(){
    var obj={
            dest:$.urlParam("dest"),
            text:document.getElementById("sendpriv").value
          };
    var query=$.ajax({
      url : '/private',
      method : 'POST',
      data : JSON.stringify(obj)
    });

  });

	$('input[name=down]').click(function(){
    var obj={
            name: document.getElementById("filename").value,
            dest: $.urlParam("dest"),
            hash: document.getElementById("metahash").value
          };
    var query=$.ajax({
      url : '/download',
      method : 'POST',
      data : JSON.stringify(obj)
    });

  });


});
