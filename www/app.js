function format_fname(address) {
    var name = address["Name"]
    if (!name) {
	return address["Address"].split("@")[0];
    } else if (name.indexOf(", ") !== -1) {
	return name.split(", ")[1];
    } else {
	return name.split(" ") [0];
    }
}

function Mailbox ($scope, $http) {
    $scope.mailboxes = ["Inbox", "All Mail"];
    $scope.format_subject = function (mail) {
	return mail["Subject"].replace(/^(Re:|Fwd:)+ /, "");
    };
    $scope.format_authors = function (conversation) {
	var authors = _.map(conversation, function (mail) {
	    return format_fname(mail["From"]);
	});
	authors = _.flatten(authors);
	author_frequency = {};
	_.each(authors, function (author) {
	    if (!author_frequency[author])
		author_frequency[author] = 0;
	    author_frequency[author]++;
	});
	var sorted_authors = _.sortBy(_.uniq(authors), function (author) {
	    author_frequency[author];
	});
	return _.uniq(sorted_authors.slice(0, 3)).join(", ");
    };
    $http.get('/inbox').success(function(data) {
	$scope.conversations = data;
    });
};
