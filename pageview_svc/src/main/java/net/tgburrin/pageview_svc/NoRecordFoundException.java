package net.tgburrin.pageview_svc;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ResponseStatus;

@SuppressWarnings("serial")
@ResponseStatus(value = HttpStatus.NO_CONTENT)
public class NoRecordFoundException extends Exception {
	public NoRecordFoundException (String message) {
		super(message);
	}

	public NoRecordFoundException (String message, Throwable err) {
		super(message, err);
	}
}
