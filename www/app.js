function Mailbox ($scope, $http) {
    $scope.mailboxes = ["Inbox", "All Mail"];
    $scope.format_subject = function (mail) {
	return mail["Subject"].replace(/^(Re:|Fwd:)+ /, "");
    };
    $scope.format_authors = function (conversation) {
	var authors = _.map(conversation, function (mail) {
	    var from = mail["From"]["Name"]
	    if (!from) {
		return mail["From"]["Address"].split("@")[0];
	    } else if (from.indexOf(", ") !== -1) {
		return from.split(", ")[1];
	    } else {
		return from.split(" ") [0];
	    }
	});
	return _.uniq(authors).join(", ");
    };
    $http.get('mails.json').success(function(data) {
	$scope.conversations = data;
    });
};
