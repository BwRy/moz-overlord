$(function() {

    function mailVerified(assertion){
        $.ajax({
            type: 'POST',
            url: '/overlord/persona/verify',
            data: {assertion: assertion},
            success: function(res, status, xhr) {
                window.location = res.originalPath;
            },
            error: function(xhr, status, err) {
                alert("Login failure: " + err);
            }
        });
    }

    var personaArguments = {
        siteName: 'Overlord'
    };

    $("#login-button").bind("click", function() {
        navigator.id.get(mailVerified, personaArguments);
    });

});
