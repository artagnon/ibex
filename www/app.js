function format_fname (address) {
    var name = address["Name"];
    var email = address["Address"];
    if (!name) {
	return email.split("@")[0];
    } else if (name == "Ramkumar Ramachandra" && email == "artagnon@gmail.com") {
	return "me";
    } else if (name.indexOf(", ") !== -1) {
	return name.split(", ")[1];
    } else {
	return name.split(" ") [0];
    }
}

var ibex = angular.module('ibex', [
    'ngRoute'
]);

ibex.config(['$locationProvider', function (locationProvider) {
    locationProvider.html5Mode(true).hashPrefix('!');
}]);

ibex.config(['$routeProvider', function (routeProvider) {
    routeProvider
	.when('/', {
	    templateUrl: '/templates/mailbox.html',
	    controller: 'Mailbox'
	})
	.when('/Inbox', {
	    templateUrl: '/templates/mailbox.html',
	    controller: 'Mailbox'
	})
	.when('/AllMail', {
	    templateUrl: '/templates/mailbox.html',
	    controller: 'Mailbox'
	})
	.when('/Inbox/:ThreadID', {
	    templateUrl: '/templates/conversation.html',
	    controller: 'Conversation'
	})
	.when('/AllMail/:ThreadID', {
	    templateUrl: '/templates/conversation.html',
	    controller: 'Conversation'
	})
}]);

ibex.controller('Mailbox', ['$scope', '$rootScope', '$http', '$location', '$routeParams'
, function (scope, rootScope, http, location, routeParams) {
    rootScope.mailboxes = {"/": "Inbox", "/AllMail": "All Mail"};
    scope.format_subject = function (mail) {
	var subject = mail["Subject"].replace(/^(Re:|Fwd:)+ /, "");
	return subject.length > 80 ? subject.slice(0, 77) + "..." : subject;
    };
    scope.format_authors = function (conversation) {
	var authors = _.map(conversation, function (mail) {
	    return format_fname(mail["From"]);
	});
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
    scope.format_date = function (unixdate) {
	return moment(unixdate, "X").fromNow();
    };
    scope.get_labels = function (conversation) {
	var label_ar = _.map(conversation, function (mail) {
	    return mail["Labels"];
	});
	labels = _.intersection.apply(_, label_ar);
	return _.filter(labels, function (label) {
	    return label.indexOf("\\");
	});
    };

    rootScope.currentLocation = location.path();
    var currentLocation = rootScope.currentLocation == '/' ? '/Inbox' : rootScope.currentLocation;

    scope.goto_conversation = function (conversation) {
	return location.path(currentLocation + '/' + conversation[0]["ThreadID"]);
    };
    http.get(currentLocation + '.json').success(function (data) {
	if (!data) { return; }
	var keys = Object.keys(data).reverse();
	rootScope.conversations = _.map(keys, function (key) {
	    return [key, data[key]];
	});
    });
}]);

ibex.controller('Conversation', ['$scope', '$rootScope', '$http', '$location', '$routeParams'
, function (scope, rootScope, http, location, routeParams) {
    scope.expand_message = function (message, event) {
	http.get('/Messages/' + message["MessageID"]).success(function (data) {
	    $(event.target).text(data["Body"]);
	});
    };
    var messages = _.filter(rootScope.conversations, function (conversation) {
	if (routeParams["ThreadID"] == conversation[1][0]["ThreadID"]) {
	    return conversation[1];
	}
    });
    messages = messages[0][1];
    scope.collapsedMessages = messages.slice(0, messages.length - 1);
    var detailedMessageID = messages[messages.length - 1]["MessageID"]
    http.get('/Messages/' + detailedMessageID).success(function (data) {
	scope.detailedMessage = data;
    });
}]);
