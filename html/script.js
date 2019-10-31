
setInterval(update,1000);


function update() {

  var query=$.ajax({
    url : '/message',
    dataType: 'json'
  });
  query.done(function( data ) {
      var chatbox=document.getElementById("chat");
      chatbox.innerHTML="";

      for (m in data) {
        chatbox.innerHTML+="<strong>"+data[m]["Origin"]+"</strong>: "+data[m]["Text"]+"<br />";
        }
      chatbox.scrollTop(chatbox.height())
  });
  query.fail(function( ) {
      document.getElementById("chat").innerHTML+="Connection problem<br />";
   });

  var query=$.ajax({
    url : '/node',
    dataType: 'json'
  });
  query.done(function( data ) {
      var nodebox=document.getElementById("peers");
      nodebox.innerHTML="";

      for (m in data) {
        nodebox.innerHTML+=data[m]["IP"]+":"+data[m]["Port"]+"<br />";
        }
      node.scrollTop(node.height())
  });
  query.fail(function( ) {
      document.getElementById("peers").innerHTML="Connection problem<br />";
   });

   var query=$.ajax({
     url : '/origin',
     dataType: 'json'
   });
   query.done(function( data ) {
       var origbox=document.getElementById("origin");
       origbox.innerHTML="";

       for (m in data) {
         origbox.innerHTML+="<a href=private.html/?dest="+data[m]["Origin"]+">"+data[m]["Text"]+"</a><br />";
         }
   });
   query.fail(function( ) {
       document.getElementById("origin").innerHTML="Connection problem<br />";
    });

}

$(document).ready(function(){
  var query=$.ajax({
    url : '/id'
  });
  query.done(function( data ) {
      var idbox=document.getElementById("id");
      idbox.innerHTML="<strong>"+data+"</strong>";
  });
  query.fail(function( ) {
      document.getElementById("id").innerHTML="";
   });
  update();

  $('input[name=send]').click(function(){
    var query=$.ajax({
      url : '/message',
      method : 'POST',
      data : document.getElementById("sendmessage").value
    });

    update()
    document.getElementById("sendmessage").value=""
  });

  $('input[name=add]').click(function(){
    var query=$.ajax({
      url : '/node',
      method : 'POST',
      data : document.getElementById("addnode").value
    });

    alert("add")
    update()
    document.getElementById("addnode").value=""
  });

  $('input[name=sharefile]').click(function(){
    var fichiers = document.getElementById("selectfile").files;
    if(fichiers.length>0){
      var query=$.ajax({
        url : '/share',
        method : 'POST',
        data : fichiers[0].name
      });
      document.getElementById("selectfile").value=""

    }
  });

});
